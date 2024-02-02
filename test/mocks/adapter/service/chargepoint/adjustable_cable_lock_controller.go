// Code generated by mockery v2.36.0. DO NOT EDIT.

package mockedchargepoint

import (
	chargepoint "github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	mock "github.com/stretchr/testify/mock"
)

// AdjustableCableLockController is an autogenerated mock type for the AdjustableCableLockController type
type AdjustableCableLockController struct {
	mock.Mock
}

// ChargepointCableLockReport provides a mock function with given fields:
func (_m *AdjustableCableLockController) ChargepointCableLockReport() (*chargepoint.CableReport, error) {
	ret := _m.Called()

	var r0 *chargepoint.CableReport
	var r1 error
	if rf, ok := ret.Get(0).(func() (*chargepoint.CableReport, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *chargepoint.CableReport); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*chargepoint.CableReport)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetChargepointCableLock provides a mock function with given fields: _a0
func (_m *AdjustableCableLockController) SetChargepointCableLock(_a0 bool) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(bool) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewAdjustableCableLockController creates a new instance of AdjustableCableLockController. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewAdjustableCableLockController(t interface {
	mock.TestingT
	Cleanup(func())
}) *AdjustableCableLockController {
	mock := &AdjustableCableLockController{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}