// Code generated by mockery v2.35.3. DO NOT EDIT.

package mockedchargepoint

import mock "github.com/stretchr/testify/mock"

// AdjustableMaxCurrentController is an autogenerated mock type for the AdjustableMaxCurrentController type
type AdjustableMaxCurrentController struct {
	mock.Mock
}

// ChargepointMaxCurrentReport provides a mock function with given fields:
func (_m *AdjustableMaxCurrentController) ChargepointMaxCurrentReport() (int64, error) {
	ret := _m.Called()

	var r0 int64
	var r1 error
	if rf, ok := ret.Get(0).(func() (int64, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() int64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int64)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetChargepointMaxCurrent provides a mock function with given fields: _a0
func (_m *AdjustableMaxCurrentController) SetChargepointMaxCurrent(_a0 int64) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(int64) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewAdjustableMaxCurrentController creates a new instance of AdjustableMaxCurrentController. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewAdjustableMaxCurrentController(t interface {
	mock.TestingT
	Cleanup(func())
}) *AdjustableMaxCurrentController {
	mock := &AdjustableMaxCurrentController{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
