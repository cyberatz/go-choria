// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package streams

import (
	"time"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/api"
	"github.com/nats-io/nats.go"
)

// Value represents a value read from a bucket, it includes metadata associated with the key, bucket and specific stored value
type Value struct {
	value     []byte
	seq       uint64
	bucket    string
	key       string
	pending   uint64
	time      time.Time
	operation Operation
}

func newValueForStored(msg *api.StoredMsg, bucket string, key string) (*Value, error) {
	v := &Value{
		value:     msg.Data,
		seq:       msg.Sequence,
		time:      msg.Time.UTC(),
		bucket:    bucket,
		key:       key,
		operation: PutOperation,
	}

	if msg.Header != nil && len(msg.Header) > 0 {
		hdrs, err := decodeHeadersMsg(msg.Header)
		if err != nil {
			return nil, err
		}

		if op := hdrs.Get(opHeader); op == opDelValue {
			v.operation = DeleteOperation
		}
	}

	return v, nil
}

func newValueForMsg(msg *nats.Msg, bucket string, key string) (*Value, error) {
	v := &Value{
		value:     msg.Data,
		bucket:    bucket,
		key:       key,
		operation: PutOperation,
	}

	if msg.Header != nil && len(msg.Header) > 0 {
		if op := msg.Header.Get(opHeader); op != "" {
			if op == opDelValue {
				v.operation = DeleteOperation
			}
		}
	}

	return v, v.parseNatsMsgMetadata(msg)
}

func (v *Value) Time() time.Time  { return v.time }
func (v *Value) Bucket() string   { return v.bucket }
func (v *Value) Key() string      { return v.key }
func (v *Value) Sequence() uint64 { return v.seq }
func (v *Value) Delta() uint64    { return v.pending }

func (v *Value) parseNatsMsgMetadata(msg *nats.Msg) error {
	meta, err := jsm.ParseJSMsgMetadata(msg)
	if err != nil {
		return err
	}

	v.seq = meta.StreamSequence()
	v.time = meta.TimeStamp()
	v.pending = meta.Pending()

	return nil
}
