// Code generated by mockery v2.23.1. DO NOT EDIT.

package mockedoutlvlswitch

import (
	fimpgo "github.com/futurehomeno/fimpgo"
	fimptype "github.com/futurehomeno/fimpgo/fimptype"

	mock "github.com/stretchr/testify/mock"

	time "time"
)

// Service is an autogenerated mock type for the Service type
type Service struct {
	mock.Mock
}

// Name provides a mock function with given fields:
func (_m *Service) Name() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// SendLevelReport provides a mock function with given fields: force
func (_m *Service) SendLevelReport(force bool) (bool, error) {
	ret := _m.Called(force)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(bool) (bool, error)); ok {
		return rf(force)
	}
	if rf, ok := ret.Get(0).(func(bool) bool); ok {
		r0 = rf(force)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(bool) error); ok {
		r1 = rf(force)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SendMessage provides a mock function with given fields: message
func (_m *Service) SendMessage(message *fimpgo.FimpMessage) error {
	ret := _m.Called(message)

	var r0 error
	if rf, ok := ret.Get(0).(func(*fimpgo.FimpMessage) error); ok {
		r0 = rf(message)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetBinaryState provides a mock function with given fields: value
func (_m *Service) SetBinaryState(value bool) error {
	ret := _m.Called(value)

	var r0 error
	if rf, ok := ret.Get(0).(func(bool) error); ok {
		r0 = rf(value)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetLevel provides a mock function with given fields: value, duration
func (_m *Service) SetLevel(value int64, duration time.Duration) error {
	ret := _m.Called(value, duration)

	var r0 error
	if rf, ok := ret.Get(0).(func(int64, time.Duration) error); ok {
		r0 = rf(value, duration)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Specification provides a mock function with given fields:
func (_m *Service) Specification() *fimptype.Service {
	ret := _m.Called()

	var r0 *fimptype.Service
	if rf, ok := ret.Get(0).(func() *fimptype.Service); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*fimptype.Service)
		}
	}

	return r0
}

// Topic provides a mock function with given fields:
func (_m *Service) Topic() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

type mockConstructorTestingTNewService interface {
	mock.TestingT
	Cleanup(func())
}

// NewService creates a new instance of Service. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewService(t mockConstructorTestingTNewService) *Service {
	mock := &Service{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
