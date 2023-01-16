// Code generated by mockery v2.16.0. DO NOT EDIT.

package mockedroot

import mock "github.com/stretchr/testify/mock"

// Resetter is an autogenerated mock type for the Resetter type
type Resetter struct {
	mock.Mock
}

// Reset provides a mock function with given fields:
func (_m *Resetter) Reset() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewResetter interface {
	mock.TestingT
	Cleanup(func())
}

// NewResetter creates a new instance of Resetter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewResetter(t mockConstructorTestingTNewResetter) *Resetter {
	mock := &Resetter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
