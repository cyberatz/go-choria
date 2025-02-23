// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ProvisionedEvent", func() {
	Describe("newProvisionedEvent", func() {
		It("Should create the event and set options", func() {
			event := newProvisionedEvent(Component("ginkgo"))
			Expect(event.Component()).To(Equal("ginkgo"))
			Expect(event.dtype).To(Equal(Provisioned))
			Expect(event.etype).To(Equal("provisioned"))
			Expect(event.TypeString()).To(Equal("provisioned"))
		})
	})

	Describe("newProvisionedEventFromJSON", func() {
		It("Should detect invalid protocols", func() {
			_, err := newProvisionedEventFromJSON([]byte(`{"protocol":"x"}`))
			Expect(err).To(MatchError("invalid protocol 'x'"))
		})

		It("Should parse valid events", func() {
			event, err := newProvisionedEventFromJSON([]byte(`{"protocol":"io.choria.lifecycle.v1.provisioned", "component":"ginkgo"}`))
			Expect(err).ToNot(HaveOccurred())
			Expect(event.Component()).To(Equal("ginkgo"))
			Expect(event.dtype).To(Equal(Provisioned))
			Expect(event.etype).To(Equal("provisioned"))
		})
	})
})
