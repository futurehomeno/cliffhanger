// Code generated by mockery v2.38.0. DO NOT EDIT.

package mockedmediaplayer

import (
	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"

	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter/service/mediaplayer"

	"github.com/stretchr/testify/mock"
)

// Service is an autogenerated mock type for the Service type
type Service struct {
	mock.Mock
}

// Name provides a mock function with given fields:
func (_m *Service) Name() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Name")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// PublishEvent provides a mock function with given fields: event
func (_m *Service) PublishEvent(event adapter.ServiceEvent) {
	_m.Called(event)
}

// SendMessage provides a mock function with given fields: message
func (_m *Service) SendMessage(message *fimpgo.FimpMessage) error {
	ret := _m.Called(message)

	if len(ret) == 0 {
		panic("no return value specified for SendMessage")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*fimpgo.FimpMessage) error); ok {
		r0 = rf(message)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SendMetadataReport provides a mock function with given fields: force
func (_m *Service) SendMetadataReport(force bool) (bool, error) {
	ret := _m.Called(force)

	if len(ret) == 0 {
		panic("no return value specified for SendMetadataReport")
	}

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

// SendMuteReport provides a mock function with given fields: force
func (_m *Service) SendMuteReport(force bool) (bool, error) {
	ret := _m.Called(force)

	if len(ret) == 0 {
		panic("no return value specified for SendMuteReport")
	}

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

// SendPlaybackModeReport provides a mock function with given fields: force
func (_m *Service) SendPlaybackModeReport(force bool) (bool, error) {
	ret := _m.Called(force)

	if len(ret) == 0 {
		panic("no return value specified for SendPlaybackModeReport")
	}

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

// SendPlaybackReport provides a mock function with given fields: force
func (_m *Service) SendPlaybackReport(force bool) (bool, error) {
	ret := _m.Called(force)

	if len(ret) == 0 {
		panic("no return value specified for SendPlaybackReport")
	}

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

// SendVolumeReport provides a mock function with given fields: force
func (_m *Service) SendVolumeReport(force bool) (bool, error) {
	ret := _m.Called(force)

	if len(ret) == 0 {
		panic("no return value specified for SendVolumeReport")
	}

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

// SetMute provides a mock function with given fields: mute
func (_m *Service) SetMute(mute bool) error {
	ret := _m.Called(mute)

	if len(ret) == 0 {
		panic("no return value specified for SetMute")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(bool) error); ok {
		r0 = rf(mute)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetPlayback provides a mock function with given fields: action
func (_m *Service) SetPlayback(action mediaplayer.PlaybackAction) error {
	ret := _m.Called(action)

	if len(ret) == 0 {
		panic("no return value specified for SetPlayback")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(mediaplayer.PlaybackAction) error); ok {
		r0 = rf(action)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetPlaybackMode provides a mock function with given fields: mode
func (_m *Service) SetPlaybackMode(mode map[string]bool) error {
	ret := _m.Called(mode)

	if len(ret) == 0 {
		panic("no return value specified for SetPlaybackMode")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(map[string]bool) error); ok {
		r0 = rf(mode)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetVolume provides a mock function with given fields: level
func (_m *Service) SetVolume(level int64) error {
	ret := _m.Called(level)

	if len(ret) == 0 {
		panic("no return value specified for SetVolume")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(int64) error); ok {
		r0 = rf(level)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Specification provides a mock function with given fields:
func (_m *Service) Specification() *fimptype.Service {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Specification")
	}

	var r0 *fimptype.Service
	if rf, ok := ret.Get(0).(func() *fimptype.Service); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*fimptype.Service)
		}
	}

	return r0
}

// Topic provides a mock function with given fields:
func (_m *Service) Topic() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Topic")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// NewService creates a new instance of Service. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewService(t interface {
	mock.TestingT
	Cleanup(func())
}) *Service {
	mock := &Service{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
