// Code generated by mockery v2.36.0. DO NOT EDIT.

package mockedadapter

import (
	adapter "github.com/futurehomeno/cliffhanger/adapter"
	mock "github.com/stretchr/testify/mock"
)

// Connector is an autogenerated mock type for the Connector type
type Connector struct {
	mock.Mock
}

// Connectivity provides a mock function with given fields:
func (_m *Connector) Connectivity() *adapter.ConnectivityDetails {
	ret := _m.Called()

	var r0 *adapter.ConnectivityDetails
	if rf, ok := ret.Get(0).(func() *adapter.ConnectivityDetails); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*adapter.ConnectivityDetails)
		}
	}

	return r0
}

// Ping provides a mock function with given fields:
func (_m *Connector) Ping() *adapter.PingDetails {
	ret := _m.Called()

	var r0 *adapter.PingDetails
	if rf, ok := ret.Get(0).(func() *adapter.PingDetails); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*adapter.PingDetails)
		}
	}

	return r0
}

// NewConnector creates a new instance of Connector. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewConnector(t interface {
	mock.TestingT
	Cleanup(func())
}) *Connector {
	mock := &Connector{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
