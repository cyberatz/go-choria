// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package execwatcher

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/ExecWatcher")
}

var _ = Describe("ExecWatcher", func() {
	var (
		mockctl     *gomock.Controller
		mockMachine *model.MockMachine
		watch       *Watcher
		now         time.Time
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		mockMachine = model.NewMockMachine(mockctl)

		mockMachine.EXPECT().Name().Return("exec").AnyTimes()
		mockMachine.EXPECT().Identity().Return("ginkgo").AnyTimes()
		mockMachine.EXPECT().InstanceID().Return("1234567890").AnyTimes()
		mockMachine.EXPECT().Version().Return("1.0.0").AnyTimes()
		mockMachine.EXPECT().TimeStampSeconds().Return(now.Unix()).AnyTimes()
		mockMachine.EXPECT().Debugf(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		mockMachine.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		now = time.Unix(1606924953, 0)

		wi, err := New(mockMachine, "ginkgo", []string{"always"}, "fail", "success", "2m", time.Second, map[string]interface{}{
			"command": "foo",
		})
		Expect(err).ToNot(HaveOccurred())
		watch = wi.(*Watcher)
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("processTemplate", func() {
		It("Should handle absent data correctly", func() {
			mockMachine.EXPECT().Facts().Return([]byte(`{"location":"lon"}`)).AnyTimes()
			mockMachine.EXPECT().Data().Return(map[string]interface{}{"x": 1}).AnyTimes()

			res, err := watch.ProcessTemplate(`{{lookup "facts.foo" "default"}}`)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal("default"))
		})

		It("Should handle present data correctly", func() {
			mockMachine.EXPECT().Facts().Return([]byte(`{"location":"lon", "i":1}`))
			mockMachine.EXPECT().Data().Return(map[string]interface{}{"x": 1})

			res, err := watch.ProcessTemplate(`{{lookup "facts.location" "default"| ToUpper }}: {{ lookup "i" 1 }} ({{ lookup "data.x" 2 }})`)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal("LON: 1 (1)"))
		})
	})

	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			prop := map[string]interface{}{
				"command":                   "cmd",
				"timeout":                   "1.5s",
				"environment":               []string{"key1=val1", "key2=val2"},
				"suppress_success_announce": "true",
			}
			Expect(watch.setProperties(prop)).ToNot(HaveOccurred())
			Expect(watch.properties.Command).To(Equal("cmd"))
			Expect(watch.properties.Timeout).To(Equal(1500 * time.Millisecond))
			Expect(watch.properties.Environment).To(Equal([]string{"key1=val1", "key2=val2"}))
			Expect(watch.properties.SuppressSuccessAnnounce).To(BeTrue())
		})

		It("Should handle errors", func() {
			watch.properties = nil
			err := watch.setProperties(map[string]interface{}{})
			Expect(err).To(MatchError("command is required"))
		})

		It("Should enforce 1 second intervals", func() {
			err := watch.setProperties(map[string]interface{}{
				"command": "cmd",
				"timeout": "0",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(watch.properties.Command).To(Equal("cmd"))
			Expect(watch.properties.Timeout).To(Equal(time.Second))
		})
	})

	Describe("CurrentState", func() {
		It("Should be a valid state", func() {
			watch.properties.Command = "/bin/sh"
			watch.previous = Success
			watch.previousRunTime = time.Second

			cs := watch.CurrentState()
			csj, err := cs.(*StateNotification).JSON()
			Expect(err).ToNot(HaveOccurred())

			event := map[string]interface{}{}
			err = json.Unmarshal(csj, &event)
			Expect(err).ToNot(HaveOccurred())
			delete(event, "id")

			Expect(event).To(Equal(map[string]interface{}{
				"time":            "2020-12-02T16:02:33Z",
				"type":            "io.choria.machine.watcher.exec.v1.state",
				"subject":         "ginkgo",
				"specversion":     "1.0",
				"source":          "io.choria.machine",
				"datacontenttype": "application/json",
				"data": map[string]interface{}{
					"command":           "/bin/sh",
					"previous_outcome":  "success",
					"previous_run_time": float64(time.Second),
					"id":                "1234567890",
					"identity":          "ginkgo",
					"machine":           "exec",
					"name":              "ginkgo",
					"protocol":          "io.choria.machine.watcher.exec.v1.state",
					"type":              "exec",
					"version":           "1.0.0",
					"timestamp":         float64(now.Unix()),
				},
			}))
		})
	})
})
