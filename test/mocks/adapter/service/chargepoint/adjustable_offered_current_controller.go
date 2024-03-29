// Code generated by mockery v2.36.0. DO NOT EDIT.

package mockedchargepoint

import mock "github.com/stretchr/testify/mock"

// AdjustableOfferedCurrentController is an autogenerated mock type for the AdjustableOfferedCurrentController type
type AdjustableOfferedCurrentController struct {
	mock.Mock
}

// SetChargepointOfferedCurrent provides a mock function with given fields: _a0
func (_m *AdjustableOfferedCurrentController) SetChargepointOfferedCurrent(_a0 int64) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(int64) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewAdjustableOfferedCurrentController creates a new instance of AdjustableOfferedCurrentController. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewAdjustableOfferedCurrentController(t interface {
	mock.TestingT
	Cleanup(func())
}) *AdjustableOfferedCurrentController {
	mock := &AdjustableOfferedCurrentController{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
