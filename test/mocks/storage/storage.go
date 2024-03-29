// Code generated by mockery v2.36.0. DO NOT EDIT.

package mockedstorage

import mock "github.com/stretchr/testify/mock"

// Storage is an autogenerated mock type for the Storage type
type Storage[T interface{}] struct {
	mock.Mock
}

// Load provides a mock function with given fields:
func (_m *Storage[T]) Load() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Model provides a mock function with given fields:
func (_m *Storage[T]) Model() T {
	ret := _m.Called()

	var r0 T
	if rf, ok := ret.Get(0).(func() T); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(T)
	}

	return r0
}

// Reset provides a mock function with given fields:
func (_m *Storage[T]) Reset() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Save provides a mock function with given fields:
func (_m *Storage[T]) Save() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewStorage creates a new instance of Storage. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewStorage[T interface{}](t interface {
	mock.TestingT
	Cleanup(func())
}) *Storage[T] {
	mock := &Storage[T]{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
