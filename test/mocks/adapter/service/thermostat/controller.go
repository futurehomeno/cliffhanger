// Code generated by mockery v2.10.0. DO NOT EDIT.

package mockedthermostat

import mock "github.com/stretchr/testify/mock"

// Controller is an autogenerated mock type for the Controller type
type Controller struct {
	mock.Mock
}

// SetThermostatMode provides a mock function with given fields: mode
func (_m *Controller) SetThermostatMode(mode string) error {
	ret := _m.Called(mode)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(mode)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetThermostatSetpoint provides a mock function with given fields: mode, value, unit
func (_m *Controller) SetThermostatSetpoint(mode string, value float64, unit string) error {
	ret := _m.Called(mode, value, unit)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, float64, string) error); ok {
		r0 = rf(mode, value, unit)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ThermostatModeReport provides a mock function with given fields:
func (_m *Controller) ThermostatModeReport() (string, error) {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ThermostatSetpointReport provides a mock function with given fields: mode
func (_m *Controller) ThermostatSetpointReport(mode string) (float64, string, error) {
	ret := _m.Called(mode)

	var r0 float64
	if rf, ok := ret.Get(0).(func(string) float64); ok {
		r0 = rf(mode)
	} else {
		r0 = ret.Get(0).(float64)
	}

	var r1 string
	if rf, ok := ret.Get(1).(func(string) string); ok {
		r1 = rf(mode)
	} else {
		r1 = ret.Get(1).(string)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string) error); ok {
		r2 = rf(mode)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// ThermostatStateReport provides a mock function with given fields:
func (_m *Controller) ThermostatStateReport() (string, error) {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}