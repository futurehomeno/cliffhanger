// Code generated by mockery v2.13.1. DO NOT EDIT.

package mockedadapter

import mock "github.com/stretchr/testify/mock"

// ThingState is an autogenerated mock type for the ThingState type
type ThingState struct {
	mock.Mock
}

// Address provides a mock function with given fields:
func (_m *ThingState) Address() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// ID provides a mock function with given fields:
func (_m *ThingState) ID() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// Info provides a mock function with given fields: model
func (_m *ThingState) Info(model interface{}) error {
	ret := _m.Called(model)

	var r0 error
	if rf, ok := ret.Get(0).(func(interface{}) error); ok {
		r0 = rf(model)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetState provides a mock function with given fields: model
func (_m *ThingState) SetState(model interface{}) error {
	ret := _m.Called(model)

	var r0 error
	if rf, ok := ret.Get(0).(func(interface{}) error); ok {
		r0 = rf(model)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// State provides a mock function with given fields: model
func (_m *ThingState) State(model interface{}) error {
	ret := _m.Called(model)

	var r0 error
	if rf, ok := ret.Get(0).(func(interface{}) error); ok {
		r0 = rf(model)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewThingState interface {
	mock.TestingT
	Cleanup(func())
}

// NewThingState creates a new instance of ThingState. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewThingState(t mockConstructorTestingTNewThingState) *ThingState {
	mock := &ThingState{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
