// Code generated by mockery v2.35.3. DO NOT EDIT.

package mockeddevsys

import mock "github.com/stretchr/testify/mock"

// RebootController is an autogenerated mock type for the RebootController type
type RebootController struct {
	mock.Mock
}

// RebootDevice provides a mock function with given fields: hard
func (_m *RebootController) RebootDevice(hard bool) error {
	ret := _m.Called(hard)

	var r0 error
	if rf, ok := ret.Get(0).(func(bool) error); ok {
		r0 = rf(hard)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewRebootController creates a new instance of RebootController. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewRebootController(t interface {
	mock.TestingT
	Cleanup(func())
}) *RebootController {
	mock := &RebootController{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
