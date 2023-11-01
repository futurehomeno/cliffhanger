// Code generated by mockery v2.35.3. DO NOT EDIT.

package mockednumericmeter

import (
	numericmeter "github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	mock "github.com/stretchr/testify/mock"
)

// Reporter is an autogenerated mock type for the Reporter type
type Reporter struct {
	mock.Mock
}

// MeterReport provides a mock function with given fields: unit
func (_m *Reporter) MeterReport(unit numericmeter.Unit) (float64, error) {
	ret := _m.Called(unit)

	var r0 float64
	var r1 error
	if rf, ok := ret.Get(0).(func(numericmeter.Unit) (float64, error)); ok {
		return rf(unit)
	}
	if rf, ok := ret.Get(0).(func(numericmeter.Unit) float64); ok {
		r0 = rf(unit)
	} else {
		r0 = ret.Get(0).(float64)
	}

	if rf, ok := ret.Get(1).(func(numericmeter.Unit) error); ok {
		r1 = rf(unit)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewReporter creates a new instance of Reporter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewReporter(t interface {
	mock.TestingT
	Cleanup(func())
}) *Reporter {
	mock := &Reporter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
