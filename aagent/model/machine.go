// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"encoding/json"

	"github.com/choria-io/go-choria/lifecycle"
	"github.com/nats-io/jsm.go"
)

type MachineConstructor interface {
	Name() string
	Machine() interface{}
	PluginName() string
}

type Machine interface {
	State() string
	Transition(t string, args ...interface{}) error
	NotifyWatcherState(string, interface{})
	Name() string
	Directory() string
	TextFileDirectory() string
	Identity() string
	InstanceID() string
	Version() string
	TimeStampSeconds() int64
	OverrideData() ([]byte, error)
	ChoriaStatusFile() (string, int)
	JetStreamConnection() (*jsm.Manager, error)
	PublishLifecycleEvent(t lifecycle.Type, opts ...lifecycle.Option)
	MainCollective() string
	Facts() json.RawMessage
	Data() map[string]interface{}
	DataPut(key string, val interface{}) error
	DataGet(key string) (interface{}, bool)
	DataDelete(key string) error
	Debugf(name string, format string, args ...interface{})
	Infof(name string, format string, args ...interface{})
	Warnf(name string, format string, args ...interface{})
	Errorf(name string, format string, args ...interface{})
}
