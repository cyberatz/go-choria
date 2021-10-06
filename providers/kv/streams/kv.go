// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package streams

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/choria-io/go-choria/inter"
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

type Operation int

const (
	allBucketSubjects = "%s.kv.>"
	listenSubjPattern = "%s.kv.%s.>"
	prefixSubjPattern = "%s.kv.%s"
	keySubjPattern    = "%s.%%s"
	streamPattern     = "KV_%s_%s"
	streamPrefix      = "KV_%s_" // to detect bucket name
	opHeader          = "Choria-Op"
	opDelValue        = "RM"

	UnknownOperation = 0
	PutOperation     = 1
	DeleteOperation  = 2
)

// StreamsKV is a Key-Value store built on Choria Streams
type StreamsKV struct {
	name       string
	collective string
	sPrefix    string
	kFormat    string
	listen     string
	stream     string
	timeout    time.Duration
	log        *logrus.Entry
	nc         *nats.Conn
	jsm        *jsm.Manager

	cfg *Config

	fw   inter.Framework
	conn inter.Connector
}

type Config struct {
	History       int
	Replicas      int
	TTL           time.Duration
	MaxBucketSize int
}

// NewStreamsKV connects to a KV Bucket, if create is false fails when it does not exist
func NewStreamsKV(ctx context.Context, name string, create bool, fw inter.Framework, opts ...Option) (*StreamsKV, error) {
	kv := &StreamsKV{
		name:       name,
		collective: fw.Configuration().MainCollective,
		fw:         fw,
		log:        fw.Logger("kv").WithField("bucket", name),
		timeout:    2 * time.Second,
		cfg: &Config{
			History:       5,
			Replicas:      1,
			MaxBucketSize: -1,
		},
	}

	for _, opt := range opts {
		opt(kv)
	}

	// mcollective.kv.cfg.>
	kv.listen = fmt.Sprintf(listenSubjPattern, kv.collective, name)
	// mcollective.kv.cfg
	kv.sPrefix = fmt.Sprintf(prefixSubjPattern, kv.collective, name)
	// mcollective.kv.cfg.%s
	kv.kFormat = fmt.Sprintf(keySubjPattern, kv.sPrefix)
	// KV_MCOLLECTIVE_cfg
	kv.stream = fmt.Sprintf(streamPattern, strings.ToUpper(kv.collective), name)

	// TODO: create

	return kv, kv.connect(ctx)
}

func (kv *StreamsKV) connect(ctx context.Context) error {
	var err error

	if kv.conn != nil {
		kv.nc = kv.conn.Nats()
		kv.jsm, err = jsm.New(kv.nc, jsm.WithTimeout(kv.timeout))
		return err
	}

	kv.log.Infof("attempting to create new connection")

	kv.conn, err = kv.fw.NewConnector(ctx, kv.fw.MiddlewareServers, fmt.Sprintf("kv %s", kv.fw.CallerID()), kv.log)
	if err != nil {
		return err
	}

	return nil
}

func (kv *StreamsKV) subjForKey(k string) string {
	return fmt.Sprintf(kv.kFormat, k)
}

// Put stores a value in the bucket returning the unique per value sequence
func (kv *StreamsKV) Put(k string, v []byte) (uint64, error) {
	res, err := kv.nc.Request(kv.subjForKey(k), v, kv.timeout)
	if err != nil {
		return 0, fmt.Errorf("put failed: %s", err)
	}

	ack, err := jsm.ParsePubAck(res)
	if err != nil {
		return 0, fmt.Errorf("put failed: %s", err)
	}

	return ack.Sequence, nil
}

// Get loads a value from the stream
func (kv *StreamsKV) Get(k string) (*Value, error) {
	res, err := kv.jsm.ReadLastMessageForSubject(kv.stream, kv.subjForKey(k))
	if err != nil {
		return nil, fmt.Errorf("unknown key")
	}

	v, err := newValueForStored(res, kv.name, k)
	if err != nil {
		return nil, fmt.Errorf("corrupt value: %s", err)
	}

	if v.operation == DeleteOperation {
		return nil, fmt.Errorf("key not found")
	}

	return v, nil
}

// Delete deletes a message in a way that preserves history
func (kv *StreamsKV) Delete(k string) (uint64, error) {
	msg := nats.NewMsg(kv.subjForKey(k))
	msg.Header.Add(opHeader, opDelValue)

	res, err := kv.nc.RequestMsg(msg, kv.timeout)
	if err != nil {
		return 0, fmt.Errorf("delete failed: %s", err)
	}

	ack, err := jsm.ParsePubAck(res)
	if err != nil {
		return 0, fmt.Errorf("delete failed: %s", err)
	}

	return ack.Sequence, nil
}

// Destroy destroys a bucket, all values are lost and the kv instance can not be used after
func (kv *StreamsKV) Destroy() error {
	s, err := kv.jsm.LoadStream(kv.stream)
	if err != nil {
		return fmt.Errorf("destroy failed: %s", err)
	}

	return s.Delete()
}

// Status retrieves the status of the bucket
func (kv *StreamsKV) Status() (*Status, error) {
	s, err := kv.jsm.LoadStream(kv.stream)
	if err != nil {
		return nil, fmt.Errorf("loading bucket status failed: %s", err)
	}

	status := &Status{bucket: kv.name}
	status.si, err = s.LatestInformation()
	if err != nil {
		return nil, fmt.Errorf("loading bucket status failed: %s", err)
	}

	return status, nil
}

// Backup creates a backup of the stream to target, returning the bytes received
func (kv *StreamsKV) Backup(ctx context.Context, targetDirectory string) (int64, error) {
	s, err := kv.jsm.LoadStream(kv.stream)
	if err != nil {
		return 0, fmt.Errorf("backup failed: %s", err)
	}

	fp, err := s.SnapshotToDirectory(ctx, targetDirectory, jsm.SnapshotHealthCheck())
	if err != nil {
		return 0, fmt.Errorf("backup failed: %s", err)
	}

	return int64(fp.BytesReceived()), nil
}

// EachBucket iterates each bucket, if the callback returns true itteration will be stopped
func EachBucket(ctx context.Context, fw inter.Framework, cb func(kv *StreamsKV) (stop bool)) error {
	conn, err := fw.NewConnector(ctx, fw.MiddlewareServers, "kv manager", fw.Logger("kv"))
	if err != nil {
		return err
	}

	mgr, err := jsm.New(conn.Nats())
	if err != nil {
		return err
	}

	mc := fw.Configuration().MainCollective

	names, err := mgr.StreamNames(&jsm.StreamNamesFilter{
		Subject: fmt.Sprintf(allBucketSubjects, mc),
	})
	if err != nil {
		return err
	}

	prefix := fmt.Sprintf(streamPrefix, mc)

	for _, name := range names {
		bucket := strings.TrimPrefix(name, prefix)
		if bucket == "" {
			continue
		}

		kv, err := NewStreamsKV(ctx, bucket, false, fw, WithConnection(conn))
		if err != nil {
			return err
		}

		if cb(kv) {
			break
		}
	}

	return nil
}

// Restore restores a backup of a bucket
func Restore(ctx context.Context, fw inter.Framework, store string) (int64, error) {
	bj, err := os.ReadFile(filepath.Join(store, "backup.json"))
	if err != nil {
		return 0, fmt.Errorf("could not read backup configuration: %s", err)
	}

	res := gjson.GetBytes(bj, "config.name")
	if !res.Exists() {
		return 0, fmt.Errorf("cannot determine bucket name from backup")
	}

	conn, err := fw.NewConnector(ctx, fw.MiddlewareServers, fmt.Sprintf("kv %s", fw.CallerID()), fw.Logger("kv"))
	if err != nil {
		return 0, err
	}

	mgr, err := jsm.New(conn.Nats())
	if err != nil {
		return 0, err
	}

	rp, _, err := mgr.RestoreSnapshotFromDirectory(ctx, res.String(), store)
	if err != nil {
		return 0, err
	}

	return int64(rp.BytesSent()), nil
}
