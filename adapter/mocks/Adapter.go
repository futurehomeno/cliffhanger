// Code generated by mockery v2.9.4. DO NOT EDIT.

package mocks

import (
	adapter "github.com/futurehomeno/cliffhanger/adapter"
	mock "github.com/stretchr/testify/mock"
)

// Adapter is an autogenerated mock type for the Adapter type
type Adapter struct {
	mock.Mock
}

// AddThing provides a mock function with given fields: thing
func (_m *Adapter) AddThing(thing adapter.Thing) error {
	ret := _m.Called(thing)

	var r0 error
	if rf, ok := ret.Get(0).(func(adapter.Thing) error); ok {
		r0 = rf(thing)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Address provides a mock function with given fields:
func (_m *Adapter) Address() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// Name provides a mock function with given fields:
func (_m *Adapter) Name() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// RegisterThing provides a mock function with given fields: thing
func (_m *Adapter) RegisterThing(thing adapter.Thing) {
	_m.Called(thing)
}

// RemoveAllThings provides a mock function with given fields:
func (_m *Adapter) RemoveAllThings() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RemoveThing provides a mock function with given fields: address
func (_m *Adapter) RemoveThing(address string) error {
	ret := _m.Called(address)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(address)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SendExclusionReport provides a mock function with given fields: thing
func (_m *Adapter) SendExclusionReport(thing adapter.Thing) error {
	ret := _m.Called(thing)

	var r0 error
	if rf, ok := ret.Get(0).(func(adapter.Thing) error); ok {
		r0 = rf(thing)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SendInclusionReport provides a mock function with given fields: thing
func (_m *Adapter) SendInclusionReport(thing adapter.Thing) error {
	ret := _m.Called(thing)

	var r0 error
	if rf, ok := ret.Get(0).(func(adapter.Thing) error); ok {
		r0 = rf(thing)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ServiceByTopic provides a mock function with given fields: topic
func (_m *Adapter) ServiceByTopic(topic string) adapter.Service {
	ret := _m.Called(topic)

	var r0 adapter.Service
	if rf, ok := ret.Get(0).(func(string) adapter.Service); ok {
		r0 = rf(topic)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(adapter.Service)
		}
	}

	return r0
}

// Services provides a mock function with given fields: name
func (_m *Adapter) Services(name string) []adapter.Service {
	ret := _m.Called(name)

	var r0 []adapter.Service
	if rf, ok := ret.Get(0).(func(string) []adapter.Service); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]adapter.Service)
		}
	}

	return r0
}

// ThingByAddress provides a mock function with given fields: address
func (_m *Adapter) ThingByAddress(address string) adapter.Thing {
	ret := _m.Called(address)

	var r0 adapter.Thing
	if rf, ok := ret.Get(0).(func(string) adapter.Thing); ok {
		r0 = rf(address)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(adapter.Thing)
		}
	}

	return r0
}

// ThingByTopic provides a mock function with given fields: topic
func (_m *Adapter) ThingByTopic(topic string) adapter.Thing {
	ret := _m.Called(topic)

	var r0 adapter.Thing
	if rf, ok := ret.Get(0).(func(string) adapter.Thing); ok {
		r0 = rf(topic)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(adapter.Thing)
		}
	}

	return r0
}

// Things provides a mock function with given fields:
func (_m *Adapter) Things() []adapter.Thing {
	ret := _m.Called()

	var r0 []adapter.Thing
	if rf, ok := ret.Get(0).(func() []adapter.Thing); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]adapter.Thing)
		}
	}

	return r0
}

// UnregisterAllThings provides a mock function with given fields:
func (_m *Adapter) UnregisterAllThings() {
	_m.Called()
}

// UnregisterThing provides a mock function with given fields: address
func (_m *Adapter) UnregisterThing(address string) {
	_m.Called(address)
}
