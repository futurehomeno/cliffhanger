// Code generated by mockery v2.10.0. DO NOT EDIT.

package mockedchargepoint

import mock "github.com/stretchr/testify/mock"

// Controller is an autogenerated mock type for the Controller type
type Controller struct {
	mock.Mock
}

// ChargepointCableLockReport provides a mock function with given fields:
func (_m *Controller) ChargepointCableLockReport() (bool, error) {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ChargepointCurrentSessionReport provides a mock function with given fields:
func (_m *Controller) ChargepointCurrentSessionReport() (float64, error) {
	ret := _m.Called()

	var r0 float64
	if rf, ok := ret.Get(0).(func() float64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(float64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ChargepointStateReport provides a mock function with given fields:
func (_m *Controller) ChargepointStateReport() (string, error) {
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

// SetChargepointCableLock provides a mock function with given fields: _a0
func (_m *Controller) SetChargepointCableLock(_a0 bool) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(bool) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StartChargepointCharging provides a mock function with given fields:
func (_m *Controller) StartChargepointCharging() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
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