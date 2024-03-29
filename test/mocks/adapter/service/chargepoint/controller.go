// Code generated by mockery v2.36.0. DO NOT EDIT.

package mockedchargepoint

import (
	chargepoint "github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	mock "github.com/stretchr/testify/mock"
)

// Controller is an autogenerated mock type for the Controller type
type Controller struct {
	mock.Mock
}

// ChargepointCurrentSessionReport provides a mock function with given fields:
func (_m *Controller) ChargepointCurrentSessionReport() (*chargepoint.SessionReport, error) {
	ret := _m.Called()

	var r0 *chargepoint.SessionReport
	var r1 error
	if rf, ok := ret.Get(0).(func() (*chargepoint.SessionReport, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *chargepoint.SessionReport); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*chargepoint.SessionReport)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ChargepointStateReport provides a mock function with given fields:
func (_m *Controller) ChargepointStateReport() (chargepoint.State, error) {
	ret := _m.Called()

	var r0 chargepoint.State
	var r1 error
	if rf, ok := ret.Get(0).(func() (chargepoint.State, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() chargepoint.State); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(chargepoint.State)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StartChargepointCharging provides a mock function with given fields: settings
func (_m *Controller) StartChargepointCharging(settings *chargepoint.ChargingSettings) error {
	ret := _m.Called(settings)

	var r0 error
	if rf, ok := ret.Get(0).(func(*chargepoint.ChargingSettings) error); ok {
		r0 = rf(settings)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StopChargepointCharging provides a mock function with given fields:
func (_m *Controller) StopChargepointCharging() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
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
