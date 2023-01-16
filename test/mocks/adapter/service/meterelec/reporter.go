// Code generated by mockery v2.16.0. DO NOT EDIT.

package mockedmeterelec

import mock "github.com/stretchr/testify/mock"

// Reporter is an autogenerated mock type for the Reporter type
type Reporter struct {
	mock.Mock
}

// ElectricityMeterReport provides a mock function with given fields: unit
func (_m *Reporter) ElectricityMeterReport(unit string) (float64, error) {
	ret := _m.Called(unit)

	var r0 float64
	if rf, ok := ret.Get(0).(func(string) float64); ok {
		r0 = rf(unit)
	} else {
		r0 = ret.Get(0).(float64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(unit)
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
