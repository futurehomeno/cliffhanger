// Code generated by mockery v2.32.4. DO NOT EDIT.

package mockeddevsys

import mock "github.com/stretchr/testify/mock"

// Controller is an autogenerated mock type for the Controller type
type Controller struct {
	mock.Mock
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
