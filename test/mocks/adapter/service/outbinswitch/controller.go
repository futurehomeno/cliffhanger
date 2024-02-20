// Code generated by mockery v2.38.0. DO NOT EDIT.

package mockedoutbinswitch

import mock "github.com/stretchr/testify/mock"

// Controller is an autogenerated mock type for the Controller type
type Controller struct {
	mock.Mock
}

// BinarySwitchStateReport provides a mock function with given fields:
func (_m *Controller) BinarySwitchStateReport() (bool, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for BinarySwitchStateReport")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func() (bool, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetBinarySwitchState provides a mock function with given fields: _a0
func (_m *Controller) SetBinarySwitchState(_a0 bool) error {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for SetBinarySwitchState")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(bool) error); ok {
		r0 = rf(_a0)
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
