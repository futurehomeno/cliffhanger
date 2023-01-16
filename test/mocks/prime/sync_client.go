// Code generated by mockery v2.16.0. DO NOT EDIT.

package mockedprime

import (
	fimpgo "github.com/futurehomeno/fimpgo"
	mock "github.com/stretchr/testify/mock"
)

// SyncClient is an autogenerated mock type for the SyncClient type
type SyncClient struct {
	mock.Mock
}

// SendReqRespFimp provides a mock function with given fields: cmdTopic, responseTopic, reqMsg, timeout, autoSubscribe
func (_m *SyncClient) SendReqRespFimp(cmdTopic string, responseTopic string, reqMsg *fimpgo.FimpMessage, timeout int64, autoSubscribe bool) (*fimpgo.FimpMessage, error) {
	ret := _m.Called(cmdTopic, responseTopic, reqMsg, timeout, autoSubscribe)

	var r0 *fimpgo.FimpMessage
	if rf, ok := ret.Get(0).(func(string, string, *fimpgo.FimpMessage, int64, bool) *fimpgo.FimpMessage); ok {
		r0 = rf(cmdTopic, responseTopic, reqMsg, timeout, autoSubscribe)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*fimpgo.FimpMessage)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, *fimpgo.FimpMessage, int64, bool) error); ok {
		r1 = rf(cmdTopic, responseTopic, reqMsg, timeout, autoSubscribe)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewSyncClient interface {
	mock.TestingT
	Cleanup(func())
}

// NewSyncClient creates a new instance of SyncClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewSyncClient(t mockConstructorTestingTNewSyncClient) *SyncClient {
	mock := &SyncClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
