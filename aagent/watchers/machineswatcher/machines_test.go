// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package machines

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMachine(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/MachinesWatcher")
}

var _ = Describe("AAgent/Watchers/MachinesWatcher", func() {
	var (
		w       *Watcher
		machine *model.MockMachine
		mockctl *gomock.Controller
		td      string
		err     error
	)

	BeforeEach(func() {
		td, err = os.MkdirTemp("", "")
		Expect(err).ToNot(HaveOccurred())
		mockctl = gomock.NewController(GinkgoT())

		machine = model.NewMockMachine(mockctl)
		machine.EXPECT().Directory().Return(td).AnyTimes()

		wi, err := New(machine, "machines", nil, "", "", "1m", time.Hour, map[string]interface{}{
			"data_item": "spec",
		})
		Expect(err).ToNot(HaveOccurred())
		w = wi.(*Watcher)
	})

	AfterEach(func() {
		mockctl.Finish()
		os.RemoveAll(td)
	})

	Describe("loadAndValidateData", func() {
		var (
			data *Specification
			pri  ed25519.PrivateKey
			pub  ed25519.PublicKey
			spec []byte
		)

		BeforeEach(func() {
			pub, pri, err = ed25519.GenerateKey(rand.Reader)
			Expect(err).ToNot(HaveOccurred())
			spec = []byte("[]")
			data = &Specification{
				Machines: []byte(base64.StdEncoding.EncodeToString(spec)),
			}
			data.Signature = hex.EncodeToString(ed25519.Sign(pri, spec))
			machine.EXPECT().DataGet(gomock.Eq("spec")).Return(data, true).AnyTimes()
		})

		It("Should function without a signature", func() {
			data.Signature = ""
			spec, err := w.loadAndValidateData()
			Expect(err).ToNot(HaveOccurred())
			Expect(spec).ToNot(BeNil())
		})

		It("Should handle data with no signatures when signature is required", func() {
			err = w.setProperties(map[string]interface{}{
				"data_item":  "spec",
				"public_key": "x",
			})
			Expect(err).ToNot(HaveOccurred())
			data.Signature = ""
			machine.EXPECT().DataDelete(gomock.Eq("spec"))
			machine.EXPECT().Errorf(gomock.Any(), gomock.Eq("No signature found in specification, removing data"))
			spec, err := w.loadAndValidateData()
			Expect(err).To(MatchError("invalid data_item"))
			Expect(spec).To(BeNil())
		})

		It("Should handle data with corrupt signatures", func() {
			err = w.setProperties(map[string]interface{}{
				"data_item":  "spec",
				"public_key": hex.EncodeToString(pub),
			})
			Expect(err).ToNot(HaveOccurred())
			data.Signature = "x"

			machine.EXPECT().DataDelete(gomock.Eq("spec"))
			machine.EXPECT().Errorf(gomock.Any(), gomock.Eq("invalid signature string, removing data %s: %s"), gomock.Eq("spec"), gomock.Any())
			spec, err := w.loadAndValidateData()
			Expect(err).To(MatchError("invalid data_item"))
			Expect(spec).To(BeNil())
		})

		It("Should handle data with invalid signatures", func() {
			err = w.setProperties(map[string]interface{}{
				"data_item":  "spec",
				"public_key": hex.EncodeToString(pub),
			})
			Expect(err).ToNot(HaveOccurred())
			data.Signature = hex.EncodeToString(ed25519.Sign(pri, []byte("wrong")))

			machine.EXPECT().DataDelete(gomock.Eq("spec"))
			machine.EXPECT().Errorf(gomock.Any(), gomock.Eq("Signature in data_item %s did not verify using configured public key '%s', removing data"), gomock.Eq("spec"), gomock.Eq(hex.EncodeToString(pub)))
			spec, err := w.loadAndValidateData()
			Expect(err).To(MatchError("invalid data_item"))
			Expect(spec).To(BeNil())
		})

		It("Should handle valid signatures", func() {
			err = w.setProperties(map[string]interface{}{
				"data_item":  "spec",
				"public_key": hex.EncodeToString(pub),
			})
			Expect(err).ToNot(HaveOccurred())

			spec, err := w.loadAndValidateData()
			Expect(err).ToNot(HaveOccurred())
			Expect(spec).To(Equal([]byte("[]")))
		})
	})
})
