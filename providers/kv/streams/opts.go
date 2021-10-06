// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package streams

import (
	"time"

	"github.com/choria-io/go-choria/inter"
)

type Option func(kv *StreamsKV)

// WithConnection sets a prepared connector to use, else a new one will be made
func WithConnection(c inter.Connector) Option {
	return func(kv *StreamsKV) { kv.conn = c }
}

// WithCollective connects to a stream in a specific collective
func WithCollective(c string) Option {
	return func(kv *StreamsKV) { kv.collective = c }
}

// WithTimeout is how long we wait for network operations
func WithTimeout(t time.Duration) Option {
	return func(kv *StreamsKV) { kv.timeout = t }
}

// BucketHistory creates a bucket with a specific amount of history kept per key
func BucketHistory(h int) Option {
	return func(kv *StreamsKV) { kv.cfg.History = h }
}

// ValueTTL will set the maximum amount of time values are kept for
func ValueTTL(ttl int) Option {
	return func(kv *StreamsKV) { kv.cfg.TTL = ttl }
}

// StorageReplicas ensures the data is replicated in the cluster
func StorageReplicas(r int) Option {
	return func(kv *StreamsKV) { kv.cfg.Replicas = r }
}

// BucketSizeLimit limits how big the bucket may be on disk, including history and metadata
func BucketSizeLimit(l uint) Option {
	return func(kv *StreamsKV) { kv.cfg.MaxBucketSize = int(l) }
}
