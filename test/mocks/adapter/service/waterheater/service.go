// Code generated by mockery v2.12.2. DO NOT EDIT.

package mockedwaterheater

import (
	fimpgo "github.com/futurehomeno/fimpgo"
	fimptype "github.com/futurehomeno/fimpgo/fimptype"

	mock "github.com/stretchr/testify/mock"

	testing "testing"
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

// SendModeReport provides a mock function with given fields: force
func (_m *Service) SendModeReport(force bool) (bool, error) {
	ret := _m.Called(force)

	var r0 bool
	if rf, ok := ret.Get(0).(func(bool) bool); ok {
		r0 = rf(force)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(bool) error); ok {
		r1 = rf(force)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SendSetpointReport provides a mock function with given fields: mode, force
func (_m *Service) SendSetpointReport(mode string, force bool) (bool, error) {
	ret := _m.Called(mode, force)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string, bool) bool); ok {
		r0 = rf(mode, force)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, bool) error); ok {
		r1 = rf(mode, force)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SendStateReport provides a mock function with given fields: force
func (_m *Service) SendStateReport(force bool) (bool, error) {
	ret := _m.Called(force)

	var r0 bool
	if rf, ok := ret.Get(0).(func(bool) bool); ok {
		r0 = rf(force)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(bool) error); ok {
		r1 = rf(force)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetMode provides a mock function with given fields: mode
func (_m *Service) SetMode(mode string) error {
	ret := _m.Called(mode)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(mode)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetSetpoint provides a mock function with given fields: mode, value, unit
func (_m *Service) SetSetpoint(mode string, value float64, unit string) error {
	ret := _m.Called(mode, value, unit)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, float64, string) error); ok {
		r0 = rf(mode, value, unit)
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

// SupportedModes provides a mock function with given fields:
func (_m *Service) SupportedModes() []string {
	ret := _m.Called()

	var r0 []string
	if rf, ok := ret.Get(0).(func() []string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}

// SupportedSetpoints provides a mock function with given fields:
func (_m *Service) SupportedSetpoints() []string {
	ret := _m.Called()

	var r0 []string
	if rf, ok := ret.Get(0).(func() []string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}

// SupportedStates provides a mock function with given fields:
func (_m *Service) SupportedStates() []string {
	ret := _m.Called()

	var r0 []string
	if rf, ok := ret.Get(0).(func() []string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}

// SupportsSetpoint provides a mock function with given fields: setpoint
func (_m *Service) SupportsSetpoint(setpoint string) bool {
	ret := _m.Called(setpoint)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(setpoint)
	} else {
		r0 = ret.Get(0).(bool)
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

// NewService creates a new instance of Service. It also registers the testing.TB interface on the mock and a cleanup function to assert the mocks expectations.
func NewService(t testing.TB) *Service {
	mock := &Service{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
