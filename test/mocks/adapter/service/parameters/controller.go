<<<<<<< HEAD
// Code generated by mockery v2.36.0. DO NOT EDIT.
=======
// Code generated by mockery v2.35.3. DO NOT EDIT.
>>>>>>> d228d6b (Changed the system to use event system.)

package mockedparameters

import (
	parameters "github.com/futurehomeno/cliffhanger/adapter/service/parameters"
	mock "github.com/stretchr/testify/mock"
)

// Controller is an autogenerated mock type for the Controller type
type Controller struct {
	mock.Mock
}

// GetParameter provides a mock function with given fields: id
func (_m *Controller) GetParameter(id string) (*parameters.Parameter, error) {
	ret := _m.Called(id)

	var r0 *parameters.Parameter
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*parameters.Parameter, error)); ok {
		return rf(id)
	}
	if rf, ok := ret.Get(0).(func(string) *parameters.Parameter); ok {
		r0 = rf(id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*parameters.Parameter)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetParameterSpecifications provides a mock function with given fields:
func (_m *Controller) GetParameterSpecifications() ([]*parameters.ParameterSpecification, error) {
	ret := _m.Called()

	var r0 []*parameters.ParameterSpecification
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]*parameters.ParameterSpecification, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []*parameters.ParameterSpecification); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*parameters.ParameterSpecification)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetParameter provides a mock function with given fields: p
func (_m *Controller) SetParameter(p *parameters.Parameter) error {
	ret := _m.Called(p)

	var r0 error
	if rf, ok := ret.Get(0).(func(*parameters.Parameter) error); ok {
		r0 = rf(p)
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
