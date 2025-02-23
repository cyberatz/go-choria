// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ShutdownEvent", func() {
	Describe("newShutdownEvent", func() {
		It("Should create the event and set options", func() {
			event := newShutdownEvent(Component("ginkgo"))
			Expect(event.Component()).To(Equal("ginkgo"))
			Expect(event.dtype).To(Equal(Shutdown))
			Expect(event.etype).To(Equal("shutdown"))
			Expect(event.TypeString()).To(Equal("shutdown"))
		})
	})

	Describe("newShutdownEventFromJSON", func() {
		It("Should detect invalid protocols", func() {
			_, err := newShutdownEventFromJSON([]byte(`{"protocol":"x"}`))
			Expect(err).To(MatchError("invalid protocol 'x'"))
		})

		It("Should parse valid events", func() {
			event, err := newShutdownEventFromJSON([]byte(`{"protocol":"io.choria.lifecycle.v1.shutdown", "component":"ginkgo"}`))
			Expect(err).ToNot(HaveOccurred())
			Expect(event.Component()).To(Equal("ginkgo"))
			Expect(event.dtype).To(Equal(Shutdown))
			Expect(event.etype).To(Equal("shutdown"))
		})
	})
})
