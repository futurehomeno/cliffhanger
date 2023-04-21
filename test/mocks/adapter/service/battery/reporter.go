// Code generated by mockery v2.23.0. DO NOT EDIT.

package mockedbattery

import (
	battery "github.com/futurehomeno/cliffhanger/adapter/service/battery"
	mock "github.com/stretchr/testify/mock"
)

// Reporter is an autogenerated mock type for the Reporter type
type Reporter struct {
	mock.Mock
}

// BatteryAlarmReport provides a mock function with given fields: event
func (_m *Reporter) BatteryAlarmReport(event string) (*battery.AlarmReport, error) {
	ret := _m.Called(event)

	var r0 *battery.AlarmReport
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*battery.AlarmReport, error)); ok {
		return rf(event)
	}
	if rf, ok := ret.Get(0).(func(string) *battery.AlarmReport); ok {
		r0 = rf(event)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*battery.AlarmReport)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(event)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// BatteryLevelReport provides a mock function with given fields:
func (_m *Reporter) BatteryLevelReport() (int64, error) {
	ret := _m.Called()

	var r0 int64
	var r1 error
	if rf, ok := ret.Get(0).(func() (int64, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() int64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int64)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
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
