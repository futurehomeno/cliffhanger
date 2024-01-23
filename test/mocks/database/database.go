// Code generated by mockery v2.35.3. DO NOT EDIT.

package mockeddatabase

import (
	time "time"

	mock "github.com/stretchr/testify/mock"
)

// Database is an autogenerated mock type for the Database type
type Database struct {
	mock.Mock
}

// Delete provides a mock function with given fields: bucket, key
func (_m *Database) Delete(bucket string, key string) error {
	ret := _m.Called(bucket, key)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(bucket, key)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Get provides a mock function with given fields: bucket, key, value
func (_m *Database) Get(bucket string, key string, value interface{}) (bool, error) {
	ret := _m.Called(bucket, key, value)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string, interface{}) (bool, error)); ok {
		return rf(bucket, key, value)
	}
	if rf, ok := ret.Get(0).(func(string, string, interface{}) bool); ok {
		r0 = rf(bucket, key, value)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(string, string, interface{}) error); ok {
		r1 = rf(bucket, key, value)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Keys provides a mock function with given fields: bucket
func (_m *Database) Keys(bucket string) ([]string, error) {
	ret := _m.Called(bucket)

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(string) ([]string, error)); ok {
		return rf(bucket)
	}
	if rf, ok := ret.Get(0).(func(string) []string); ok {
		r0 = rf(bucket)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(bucket)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// KeysBetween provides a mock function with given fields: bucket, from, to
func (_m *Database) KeysBetween(bucket string, from string, to string) ([]string, error) {
	ret := _m.Called(bucket, from, to)

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string, string) ([]string, error)); ok {
		return rf(bucket, from, to)
	}
	if rf, ok := ret.Get(0).(func(string, string, string) []string); ok {
		r0 = rf(bucket, from, to)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(string, string, string) error); ok {
		r1 = rf(bucket, from, to)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// KeysFrom provides a mock function with given fields: bucket, from
func (_m *Database) KeysFrom(bucket string, from string) ([]string, error) {
	ret := _m.Called(bucket, from)

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string) ([]string, error)); ok {
		return rf(bucket, from)
	}
	if rf, ok := ret.Get(0).(func(string, string) []string); ok {
		r0 = rf(bucket, from)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(bucket, from)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Reset provides a mock function with given fields:
func (_m *Database) Reset() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Set provides a mock function with given fields: bucket, key, value
func (_m *Database) Set(bucket string, key string, value interface{}) error {
	ret := _m.Called(bucket, key, value)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, interface{}) error); ok {
		r0 = rf(bucket, key, value)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetWithExpiry provides a mock function with given fields: bucket, key, value, expiry
func (_m *Database) SetWithExpiry(bucket string, key string, value interface{}, expiry time.Duration) error {
	ret := _m.Called(bucket, key, value, expiry)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, interface{}, time.Duration) error); ok {
		r0 = rf(bucket, key, value, expiry)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Start provides a mock function with given fields:
func (_m *Database) Start() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Stop provides a mock function with given fields:
func (_m *Database) Stop() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewDatabase creates a new instance of Database. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewDatabase(t interface {
	mock.TestingT
	Cleanup(func())
}) *Database {
	mock := &Database{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
