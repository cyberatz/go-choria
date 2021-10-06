// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package streams

import (
	"time"

	"github.com/nats-io/jsm.go/api"
)

type Status struct {
	bucket string
	si     *api.StreamInfo
}

func (s *Status) Bucket() string       { return s.bucket }
func (s *Status) Values() uint64       { return s.si.State.Msgs }
func (s *Status) Size() uint64         { return s.si.State.Bytes }
func (s *Status) History() int64       { return s.si.Config.MaxMsgsPer }
func (s *Status) TTL() time.Duration   { return s.si.Config.MaxAge }
func (s *Status) MaxBucketSize() int64 { return s.si.Config.MaxBytes }
func (s *Status) Replicas() (ok []string, failed []string) {
	if s.si.Cluster != nil {
		ok = append(ok, s.si.Cluster.Leader)
		for _, peer := range s.si.Cluster.Replicas {
			if peer.Current && !peer.Offline {
				ok = append(ok, peer.Name)
			} else {
				failed = append(failed, peer.Name)
			}
		}

	}

	return ok, failed
}
