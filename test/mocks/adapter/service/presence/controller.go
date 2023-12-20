<<<<<<< HEAD
// Code generated by mockery v2.36.0. DO NOT EDIT.
=======
// Code generated by mockery v2.35.3. DO NOT EDIT.
>>>>>>> d228d6b (Changed the system to use event system.)

package mockedpresence

import mock "github.com/stretchr/testify/mock"

// Controller is an autogenerated mock type for the Controller type
type Controller struct {
	mock.Mock
}

// SensorPresenceReport provides a mock function with given fields:
func (_m *Controller) SensorPresenceReport() (bool, error) {
	ret := _m.Called()

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
