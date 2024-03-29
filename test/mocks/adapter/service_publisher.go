// Code generated by mockery v2.36.0. DO NOT EDIT.

package mockedadapter

import (
	adapter "github.com/futurehomeno/cliffhanger/adapter"
	fimpgo "github.com/futurehomeno/fimpgo"

	mock "github.com/stretchr/testify/mock"
)

// ServicePublisher is an autogenerated mock type for the ServicePublisher type
type ServicePublisher struct {
	mock.Mock
}

// PublishServiceEvent provides a mock function with given fields: service, payload
func (_m *ServicePublisher) PublishServiceEvent(service adapter.Service, payload adapter.ServiceEvent) {
	_m.Called(service, payload)
}

// PublishServiceMessage provides a mock function with given fields: service, message
func (_m *ServicePublisher) PublishServiceMessage(service adapter.Service, message *fimpgo.FimpMessage) error {
	ret := _m.Called(service, message)

	var r0 error
	if rf, ok := ret.Get(0).(func(adapter.Service, *fimpgo.FimpMessage) error); ok {
		r0 = rf(service, message)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewServicePublisher creates a new instance of ServicePublisher. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewServicePublisher(t interface {
	mock.TestingT
	Cleanup(func())
}) *ServicePublisher {
	mock := &ServicePublisher{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
