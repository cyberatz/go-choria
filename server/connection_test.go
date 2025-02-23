// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"strings"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/config"
	imock "github.com/choria-io/go-choria/inter/imocks"
	"github.com/choria-io/go-choria/srvcache"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Server/Connection", func() {
	var _ = Describe("brokerUrls", func() {
		var (
			mockctl *gomock.Controller
			cfg     *config.Config
			fw      *imock.MockFramework
			srv     *Instance
			err     error
			ctx     context.Context
			cancel  func()
		)

		BeforeEach(func() {
			mockctl = gomock.NewController(GinkgoT())
			fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter)
			cfg.DisableTLS = true
			cfg.Choria.MiddlewareHosts = []string{"d1:4222", "d2:4222"}

			srv, err = NewInstance(fw)
			Expect(err).ToNot(HaveOccurred())

			logrus.SetLevel(logrus.FatalLevel)

			ctx, cancel = context.WithCancel(context.Background())
		})

		AfterEach(func() {
			cancel()
			mockctl.Finish()
		})

		It("Should support provisioning", func() {
			build.ProvisionModeDefault = "true"
			build.ProvisionBrokerURLs = "nats1:4222, nats2:4222"
			cfg.InitiatedByServer = true

			fw.EXPECT().ProvisionMode().Return(true).AnyTimes()
			fw.EXPECT().ProvisioningServers(gomock.Any()).Return(srvcache.StringHostsToServers(strings.Split(build.ProvisionBrokerURLs, ","), "nats"))
			servers, err := srv.brokerUrls(ctx)
			Expect(err).ToNot(HaveOccurred())

			found := servers.Servers()
			Expect(found[0].Host()).To(Equal("nats1"))
			Expect(found[1].Host()).To(Equal("nats2"))
		})

		It("Should fail gracefully for incorrect format provisioning servers", func() {
			build.ProvisionModeDefault = "true"
			build.ProvisionBrokerURLs = "invalid stuff"

			fw.EXPECT().ProvisionMode().Return(true).AnyTimes()
			fw.EXPECT().ProvisioningServers(gomock.Any()).Return(srvcache.StringHostsToServers(strings.Split(build.ProvisionBrokerURLs, ","), "nats"))
			fw.EXPECT().MiddlewareServers().Return(srvcache.StringHostsToServers(cfg.Choria.MiddlewareHosts, "nats"))
			servers, err := srv.brokerUrls(ctx)
			Expect(err).ToNot(HaveOccurred())

			found := servers.Servers()
			Expect(found).To(HaveLen(2))
			Expect(found[0].Host()).To(Equal("d1"))
			Expect(found[1].Host()).To(Equal("d2"))
		})

		It("Should fail gracefully when no servers are compiled in but provisioning is on", func() {
			build.ProvisionModeDefault = "true"
			build.ProvisionBrokerURLs = ""

			fw.EXPECT().ProvisionMode().Return(true).AnyTimes()
			fw.EXPECT().ProvisioningServers(gomock.Any()).Return(srvcache.StringHostsToServers(strings.Split(build.ProvisionBrokerURLs, ","), "nats"))
			fw.EXPECT().MiddlewareServers().Return(srvcache.StringHostsToServers(cfg.Choria.MiddlewareHosts, "nats"))

			servers, err := srv.brokerUrls(ctx)
			Expect(err).ToNot(HaveOccurred())

			found := servers.Servers()
			Expect(found).To(HaveLen(2))
			Expect(found[0].Host()).To(Equal("d1"))
			Expect(found[1].Host()).To(Equal("d2"))
		})

		It("Should default to unprovisioned mode", func() {
			fw.EXPECT().ProvisionMode().Return(false).AnyTimes()
			fw.EXPECT().MiddlewareServers().Return(srvcache.StringHostsToServers(cfg.Choria.MiddlewareHosts, "nats"))

			servers, err := srv.brokerUrls(ctx)
			Expect(err).ToNot(HaveOccurred())

			found := servers.Servers()
			Expect(found).To(HaveLen(2))
			Expect(found[0].Host()).To(Equal("d1"))
			Expect(found[1].Host()).To(Equal("d2"))
		})
	})
})
