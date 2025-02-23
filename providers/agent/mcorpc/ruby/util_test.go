// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package ruby

import (
	"io"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func Test(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Providers/Agent/McoRPC/Ruby")
}

var _ = Describe("McoRPC/Ruby", func() {
	var logger *logrus.Entry

	BeforeEach(func() {
		l := logrus.New()
		l.Out = io.Discard
		logger = l.WithFields(logrus.Fields{})
	})

	var _ = Describe("Provider", func() {
		var _ = Describe("shouldLoadAgent", func() {
			It("Should not load agents in the deny list", func() {
				Expect(shouldLoadAgent("rpcutil")).To(BeFalse())
				Expect(shouldLoadAgent("choria_util")).To(BeFalse())
				Expect(shouldLoadAgent("discovery")).To(BeFalse())
				Expect(shouldLoadAgent("foo")).To(BeTrue())
			})
		})

		var _ = Describe("loadAgents", func() {
			It("Should load only ruby agents in all libdirs", func() {
				p := Provider{
					log: logger,
				}

				p.loadAgents([]string{"testdata/lib1", "testdata/lib2"})

				Expect(p.Agents()).To(HaveLen(2))
				Expect(p.Agents()[0].Metadata.Name).To(Equal("one"))
				Expect(p.Agents()[1].Metadata.Name).To(Equal("two"))
			})
		})
	})
})
