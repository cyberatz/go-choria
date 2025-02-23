// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package natsstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/srvcache"
	"github.com/nats-io/stan.go"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/backoff"
	"github.com/choria-io/go-choria/broker/adapter/ingest"
	"github.com/choria-io/go-choria/broker/adapter/stats"
	"github.com/choria-io/go-choria/broker/adapter/transformer"
)

type stream struct {
	servers     func() (srvcache.Servers, error)
	clusterID   string
	clientID    string
	topic       string
	conn        stan.Conn
	log         *log.Entry
	name        string
	adapterName string

	work chan ingest.Adaptable
	mu   *sync.Mutex
}

func newStream(name string, work chan ingest.Adaptable, logger *log.Entry) ([]*stream, error) {
	prefix := fmt.Sprintf("plugin.choria.adapter.%s.stream.", name)

	instances, err := strconv.Atoi(cfg.Option(prefix+"workers", "10"))
	if err != nil {
		return nil, fmt.Errorf("%s should be a integer number", prefix+"workers")
	}

	servers := cfg.Option(prefix+"servers", "")
	if servers == "" {
		return nil, fmt.Errorf("no Stream servers configured, please set %s", prefix+"servers")
	}

	topic := cfg.Option(prefix+"topic", "")
	if topic == "" {
		topic = name
	}

	clusterID := cfg.Option(prefix+"clusterid", "")
	if clusterID == "" {
		return nil, fmt.Errorf("no ClusterID configured, please set %s", prefix+"clusterid'")
	}

	workers := []*stream{}

	for i := 0; i < instances; i++ {
		logger.Infof("Creating NATS Streaming Adapter %s NATS Streaming instance %d / %d publishing to %s on cluster %s", name, i, instances, topic, clusterID)

		iname := fmt.Sprintf("%s_%d-%s", name, i, strings.Replace(util.UniqueID(), "-", "", -1))

		st := &stream{
			clusterID:   clusterID,
			clientID:    iname,
			topic:       topic,
			name:        fmt.Sprintf("%s.%d", name, i),
			adapterName: name,
			work:        work,
			log:         logger.WithFields(log.Fields{"side": "stream", "instance": i}),
			mu:          &sync.Mutex{},
		}
		st.servers = st.resolver(strings.Split(servers, ","))

		workers = append(workers, st)
	}

	return workers, nil
}

func (sc *stream) resolver(parts []string) func() (srvcache.Servers, error) {
	servers, err := srvcache.StringHostsToServers(parts, "nats")
	return func() (srvcache.Servers, error) {
		return servers, err
	}
}

func (sc *stream) connect(ctx context.Context, cm inter.ConnectionManager) error {
	if ctx.Err() != nil {
		return fmt.Errorf("shutdown called")
	}

	reconn := make(chan struct{})

	nc, err := cm.NewConnector(ctx, sc.servers, sc.clientID, sc.log)
	if err != nil {
		return fmt.Errorf("could not start NATS connection: %s", err)
	}

	start := func() error {
		sc.log.Infof("%s connecting to NATS Stream", sc.clientID)

		sc.mu.Lock()
		defer sc.mu.Unlock()

		ctr := 0

		for {
			ctr++

			if ctx.Err() != nil {
				return errors.New("shutdown called")
			}

			sc.conn, err = stan.Connect(sc.clusterID, sc.clientID, stan.NatsConn(nc.Nats()), stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
				sc.log.Errorf("NATS Streaming connection got disconnected, reconnecting: %s", reason)
				stats.ErrorCtr.WithLabelValues(sc.name, "output", cfg.Identity).Inc()
				reconn <- struct{}{}
			}))
			if err != nil {
				sc.log.Errorf("Could not create initial STAN connection, retrying: %s", err)
				backoff.Default.TrySleep(ctx, ctr)

				continue
			}

			break
		}

		return nil
	}

	watcher := func() {
		ctr := 0

		for {
			select {
			case <-reconn:
				ctr++

				sc.log.WithField("attempt", ctr).Infof("Attempting to reconnect NATS Stream after reconnection")

				backoff.Default.TrySleep(ctx, ctr)

				err := start()
				if err != nil {
					sc.log.Errorf("Could not restart NATS Streaming connection: %s", err)
					reconn <- struct{}{}
				}

			case <-ctx.Done():
				return
			}
		}
	}

	err = start()
	if err != nil {
		return fmt.Errorf("could not start initial NATS Streaming connection: %s", err)
	}

	go watcher()

	sc.log.Infof("%s connected to NATS Stream", sc.clientID)

	return nil
}

func (sc *stream) disconnect() {
	if sc.conn != nil {
		sc.log.Info("Disconnecting from NATS Streaming")
		sc.conn.Close()
	}
}

func (sc *stream) publisher(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	bytes := stats.BytesCtr.WithLabelValues(sc.name, "output", cfg.Identity)
	ectr := stats.ErrorCtr.WithLabelValues(sc.name, "output", cfg.Identity)
	ctr := stats.ReceivedMsgsCtr.WithLabelValues(sc.name, "output", cfg.Identity)
	timer := stats.ProcessTime.WithLabelValues(sc.name, "output", cfg.Identity)
	workqlen := stats.WorkQueueLengthGauge.WithLabelValues(sc.adapterName, cfg.Identity)

	transformerf := func(r ingest.Adaptable) {
		obs := prometheus.NewTimer(timer)
		defer obs.ObserveDuration()
		defer func() { workqlen.Set(float64(len(sc.work))) }()

		j, err := json.Marshal(transformer.TransformToOutput(r, "natsstream"))
		if err != nil {
			sc.log.Warnf("Cannot JSON encode message for publishing to STAN, discarding: %s", err)
			ectr.Inc()
			return
		}

		sc.log.Debugf("Publishing registration data from %s to %s", r.SenderID(), sc.topic)

		bytes.Add(float64(len(j)))

		// avoids publishing during reconnects while sc.conn could be nil
		sc.mu.Lock()
		defer sc.mu.Unlock()

		err = sc.conn.Publish(sc.topic, j)
		if err != nil {
			sc.log.Warnf("Could not publish message to STAN %s, discarding: %s", sc.topic, err)
			ectr.Inc()
			return
		}

		ctr.Inc()
	}

	for {
		select {
		case r := <-sc.work:
			transformerf(r)

		case <-ctx.Done():
			sc.disconnect()

			return
		}
	}
}
