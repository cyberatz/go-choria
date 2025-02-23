// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"context"
	"sync"

	"github.com/choria-io/go-choria/config"
)

type TestReporter interface {
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

// StartTestChoriaNetwork starts a broker and instance server connected to the broker
func StartTestChoriaNetwork(t TestReporter, cfg *config.Config) (n *ChoriaNetwork) {
	n, err := StartChoriaNetwork(cfg)
	if err != nil {
		t.Fatalf("instance network failed to start: %v", err)
	}

	return n
}

// StartChoriaNetwork starts a broker anc instance server connect to the broker
func StartChoriaNetwork(cfg *config.Config) (n *ChoriaNetwork, err error) {
	n = &ChoriaNetwork{cfg: cfg}
	err = n.Start()

	return n, err
}

// StartBroker creates a new NATS broker listening on a random unused port
func StartBroker() (b *Broker, err error) {
	b = &Broker{}
	err = b.Start()

	return b, err
}

// StartChoriaServer starts an instance of instance server
func StartChoriaServer(b *Broker, cfg *config.Config) (c *ChoriaServer, err error) {
	c = &ChoriaServer{
		broker: b,
		cfg:    cfg,
		wg:     &sync.WaitGroup{},
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())

	err = c.Start()
	if err != nil {
		return nil, err
	}

	return c, nil
}
