// Code generated by mockery v2.23.0. DO NOT EDIT.

package mockednumericmeter

import (
	fimpgo "github.com/futurehomeno/fimpgo"
	fimptype "github.com/futurehomeno/fimpgo/fimptype"

	mock "github.com/stretchr/testify/mock"
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

// ResetMeter provides a mock function with given fields:
func (_m *Service) ResetMeter() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
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

// SendMeterExportReport provides a mock function with given fields: unit, force
func (_m *Service) SendMeterExportReport(unit string, force bool) (bool, error) {
	ret := _m.Called(unit, force)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(string, bool) (bool, error)); ok {
		return rf(unit, force)
	}
	if rf, ok := ret.Get(0).(func(string, bool) bool); ok {
		r0 = rf(unit, force)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(string, bool) error); ok {
		r1 = rf(unit, force)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SendMeterExtendedReport provides a mock function with given fields: values, force
func (_m *Service) SendMeterExtendedReport(values []string, force bool) (bool, error) {
	ret := _m.Called(values, force)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func([]string, bool) (bool, error)); ok {
		return rf(values, force)
	}
	if rf, ok := ret.Get(0).(func([]string, bool) bool); ok {
		r0 = rf(values, force)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func([]string, bool) error); ok {
		r1 = rf(values, force)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SendMeterReport provides a mock function with given fields: unit, force
func (_m *Service) SendMeterReport(unit string, force bool) (bool, error) {
	ret := _m.Called(unit, force)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(string, bool) (bool, error)); ok {
		return rf(unit, force)
	}
	if rf, ok := ret.Get(0).(func(string, bool) bool); ok {
		r0 = rf(unit, force)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(string, bool) error); ok {
		r1 = rf(unit, force)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
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

// SupportedExportUnits provides a mock function with given fields:
func (_m *Service) SupportedExportUnits() []string {
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

// SupportedExtendedValues provides a mock function with given fields:
func (_m *Service) SupportedExtendedValues() []string {
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

// SupportedUnits provides a mock function with given fields:
func (_m *Service) SupportedUnits() []string {
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

// SupportsExportReport provides a mock function with given fields:
func (_m *Service) SupportsExportReport() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// SupportsExtendedReport provides a mock function with given fields:
func (_m *Service) SupportsExtendedReport() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
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