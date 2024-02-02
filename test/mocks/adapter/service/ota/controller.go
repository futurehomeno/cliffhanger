// Code generated by mockery v2.36.0. DO NOT EDIT.

package mockedota

import (
	ota "github.com/futurehomeno/cliffhanger/adapter/service/ota"
	mock "github.com/stretchr/testify/mock"
)

// Controller is an autogenerated mock type for the Controller type
type Controller struct {
	mock.Mock
}

// OTAStatusReport provides a mock function with given fields:
func (_m *Controller) OTAStatusReport() (ota.StatusReport, error) {
	ret := _m.Called()

	var r0 ota.StatusReport
	var r1 error
	if rf, ok := ret.Get(0).(func() (ota.StatusReport, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() ota.StatusReport); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(ota.StatusReport)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StartOTAUpdate provides a mock function with given fields: firmwarePath
func (_m *Controller) StartOTAUpdate(firmwarePath string) error {
	ret := _m.Called(firmwarePath)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(firmwarePath)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewController creates a new instance of Controller. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewController(t interface {
	mock.TestingT
	Cleanup(func())
}) *Controller {
	mock := &Controller{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
