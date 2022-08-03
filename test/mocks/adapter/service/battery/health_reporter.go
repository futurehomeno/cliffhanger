// Code generated by mockery v2.13.1. DO NOT EDIT.

package mockedbattery

import (
	battery "github.com/futurehomeno/cliffhanger/adapter/service/battery"
	mock "github.com/stretchr/testify/mock"
)

// HealthReporter is an autogenerated mock type for the HealthReporter type
type HealthReporter struct {
	mock.Mock
}

// BatteryAlarmReport provides a mock function with given fields:
func (_m *HealthReporter) BatteryAlarmReport() (battery.AlarmReport, error) {
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
func (_m *HealthReporter) BatteryFullReport() (battery.FullReport, error) {
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
func (_m *HealthReporter) BatteryHealthReport() (int64, error) {
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
func (_m *HealthReporter) BatteryLevelReport() (int64, string, error) {
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

type mockConstructorTestingTNewHealthReporter interface {
	mock.TestingT
	Cleanup(func())
}

// NewHealthReporter creates a new instance of HealthReporter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewHealthReporter(t mockConstructorTestingTNewHealthReporter) *HealthReporter {
	mock := &HealthReporter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
