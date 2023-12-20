// Code generated by mockery v2.35.3. DO NOT EDIT.

package mockedadapter

import mock "github.com/stretchr/testify/mock"

// ThingEvent is an autogenerated mock type for the ThingEvent type
type ThingEvent struct {
	mock.Mock
}

// Address provides a mock function with given fields:
func (_m *ThingEvent) Address() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// Class provides a mock function with given fields:
func (_m *ThingEvent) Class() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// Domain provides a mock function with given fields:
func (_m *ThingEvent) Domain() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// Payload provides a mock function with given fields:
func (_m *ThingEvent) Payload() interface{} {
	ret := _m.Called()

	var r0 interface{}
	if rf, ok := ret.Get(0).(func() interface{}); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(interface{})
		}
	}

	return r0
}

// Type provides a mock function with given fields:
func (_m *ThingEvent) Type() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// NewThingEvent creates a new instance of ThingEvent. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewThingEvent(t interface {
	mock.TestingT
	Cleanup(func())
}) *ThingEvent {
	mock := &ThingEvent{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
