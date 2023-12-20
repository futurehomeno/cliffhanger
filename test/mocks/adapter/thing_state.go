<<<<<<< HEAD
// Code generated by mockery v2.36.0. DO NOT EDIT.
=======
// Code generated by mockery v2.35.3. DO NOT EDIT.
>>>>>>> d228d6b (Changed the system to use event system.)

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

// GetInclusionChecksum provides a mock function with given fields:
func (_m *ThingState) GetInclusionChecksum() uint32 {
	ret := _m.Called()

	var r0 uint32
	if rf, ok := ret.Get(0).(func() uint32); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint32)
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

// SetInclusionChecksum provides a mock function with given fields: checksum
func (_m *ThingState) SetInclusionChecksum(checksum uint32) error {
	ret := _m.Called(checksum)

	var r0 error
	if rf, ok := ret.Get(0).(func(uint32) error); ok {
		r0 = rf(checksum)
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

// NewThingState creates a new instance of ThingState. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewThingState(t interface {
	mock.TestingT
	Cleanup(func())
}) *ThingState {
	mock := &ThingState{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
