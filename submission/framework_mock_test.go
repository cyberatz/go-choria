// Code generated by MockGen. DO NOT EDIT.
// Source: disk_spool.go

// Package spool is a generated GoMock package.
package submission

import (
	reflect "reflect"

	config "github.com/choria-io/go-choria/config"
	gomock "github.com/golang/mock/gomock"
	logrus "github.com/sirupsen/logrus"
)

// MockFramework is a mock of Framework interface.
type MockFramework struct {
	ctrl     *gomock.Controller
	recorder *MockFrameworkMockRecorder
}

// MockFrameworkMockRecorder is the mock recorder for MockFramework.
type MockFrameworkMockRecorder struct {
	mock *MockFramework
}

// NewMockFramework creates a new mock instance.
func NewMockFramework(ctrl *gomock.Controller) *MockFramework {
	mock := &MockFramework{ctrl: ctrl}
	mock.recorder = &MockFrameworkMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFramework) EXPECT() *MockFrameworkMockRecorder {
	return m.recorder
}

// Configuration mocks base method.
func (m *MockFramework) Configuration() *config.Config {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Configuration")
	ret0, _ := ret[0].(*config.Config)
	return ret0
}

// Configuration indicates an expected call of Configuration.
func (mr *MockFrameworkMockRecorder) Configuration() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Configuration", reflect.TypeOf((*MockFramework)(nil).Configuration))
}

// Logger mocks base method.
func (m *MockFramework) Logger(component string) *logrus.Entry {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Logger", component)
	ret0, _ := ret[0].(*logrus.Entry)
	return ret0
}

// Logger indicates an expected call of Logger.
func (mr *MockFrameworkMockRecorder) Logger(component interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Logger", reflect.TypeOf((*MockFramework)(nil).Logger), component)
}
