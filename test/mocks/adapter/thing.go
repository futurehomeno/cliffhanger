<<<<<<< HEAD
// Code generated by mockery v2.36.0. DO NOT EDIT.
=======
// Code generated by mockery v2.35.3. DO NOT EDIT.
>>>>>>> d228d6b (Changed the system to use event system.)

package mockedadapter

import (
	adapter "github.com/futurehomeno/cliffhanger/adapter"
	fimptype "github.com/futurehomeno/fimpgo/fimptype"

	mock "github.com/stretchr/testify/mock"
)

// Thing is an autogenerated mock type for the Thing type
type Thing struct {
	mock.Mock
}

// Address provides a mock function with given fields:
func (_m *Thing) Address() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// Connect provides a mock function with given fields:
func (_m *Thing) Connect() {
	_m.Called()
}

// ConnectivityReport provides a mock function with given fields:
func (_m *Thing) ConnectivityReport() *adapter.ConnectivityReport {
	ret := _m.Called()

	var r0 *adapter.ConnectivityReport
	if rf, ok := ret.Get(0).(func() *adapter.ConnectivityReport); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*adapter.ConnectivityReport)
		}
	}

	return r0
}

// Disconnect provides a mock function with given fields:
func (_m *Thing) Disconnect() {
	_m.Called()
}

// InclusionReport provides a mock function with given fields:
func (_m *Thing) InclusionReport() *fimptype.ThingInclusionReport {
	ret := _m.Called()

	var r0 *fimptype.ThingInclusionReport
	if rf, ok := ret.Get(0).(func() *fimptype.ThingInclusionReport); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*fimptype.ThingInclusionReport)
		}
	}

	return r0
}

// SendConnectivityReport provides a mock function with given fields: force
func (_m *Thing) SendConnectivityReport(force bool) (bool, error) {
	ret := _m.Called(force)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(bool) (bool, error)); ok {
		return rf(force)
	}
	if rf, ok := ret.Get(0).(func(bool) bool); ok {
		r0 = rf(force)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(bool) error); ok {
		r1 = rf(force)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SendInclusionReport provides a mock function with given fields: force
func (_m *Thing) SendInclusionReport(force bool) (bool, error) {
	ret := _m.Called(force)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(bool) (bool, error)); ok {
		return rf(force)
	}
	if rf, ok := ret.Get(0).(func(bool) bool); ok {
		r0 = rf(force)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(bool) error); ok {
		r1 = rf(force)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SendPingReport provides a mock function with given fields:
func (_m *Thing) SendPingReport() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ServiceByTopic provides a mock function with given fields: topic
func (_m *Thing) ServiceByTopic(topic string) adapter.Service {
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
func (_m *Thing) Services(name string) []adapter.Service {
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

// Update provides a mock function with given fields: _a0
func (_m *Thing) Update(_a0 ...adapter.ThingUpdate) error {
	_va := make([]interface{}, len(_a0))
	for _i := range _a0 {
		_va[_i] = _a0[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 error
	if rf, ok := ret.Get(0).(func(...adapter.ThingUpdate) error); ok {
		r0 = rf(_a0...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewThing creates a new instance of Thing. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewThing(t interface {
	mock.TestingT
	Cleanup(func())
}) *Thing {
	mock := &Thing{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
