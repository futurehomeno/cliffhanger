// Code generated by mockery v2.13.1. DO NOT EDIT.

package mockedbattery

import (
	battery "github.com/futurehomeno/cliffhanger/adapter/service/battery"
	mock "github.com/stretchr/testify/mock"
)

// Reporter is an autogenerated mock type for the Reporter type
type Reporter struct {
	mock.Mock
}

// BatteryAlarmReport provides a mock function with given fields:
func (_m *Reporter) BatteryAlarmReport() (battery.AlarmReport, error) {
	ret := _m.Called()

	var r0 battery.AlarmReport
	if rf, ok := ret.Get(0).(func() battery.AlarmReport); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(battery.AlarmReport)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// BatteryFullReport provides a mock function with given fields:
func (_m *Reporter) BatteryFullReport() (battery.FullReport, error) {
	ret := _m.Called()

	var r0 battery.FullReport
	if rf, ok := ret.Get(0).(func() battery.FullReport); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(battery.FullReport)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// BatteryHealthReport provides a mock function with given fields:
func (_m *Reporter) BatteryHealthReport() (int64, error) {
	ret := _m.Called()

	var r0 int64
	if rf, ok := ret.Get(0).(func() int64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// BatteryLevelReport provides a mock function with given fields:
func (_m *Reporter) BatteryLevelReport() (int64, string, error) {
	ret := _m.Called()

	var r0 int64
	if rf, ok := ret.Get(0).(func() int64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int64)
	}

	var r1 string
	if rf, ok := ret.Get(1).(func() string); ok {
		r1 = rf()
	} else {
		r1 = ret.Get(1).(string)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func() error); ok {
		r2 = rf()
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// BatterySensorReport provides a mock function with given fields:
func (_m *Reporter) BatterySensorReport() (float64, string, error) {
	ret := _m.Called()

	var r0 float64
	if rf, ok := ret.Get(0).(func() float64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(float64)
	}

	var r1 string
	if rf, ok := ret.Get(1).(func() string); ok {
		r1 = rf()
	} else {
		r1 = ret.Get(1).(string)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func() error); ok {
		r2 = rf()
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

type mockConstructorTestingTNewReporter interface {
	mock.TestingT
	Cleanup(func())
}

// NewReporter creates a new instance of Reporter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewReporter(t mockConstructorTestingTNewReporter) *Reporter {
	mock := &Reporter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}