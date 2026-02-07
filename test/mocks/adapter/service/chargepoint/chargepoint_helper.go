package mockedchargepoint

import (
	"github.com/stretchr/testify/mock"
)

type MockedChargepoint struct {
	*Controller
	*AdjustableMaxCurrentController     // optional
	*AdjustableOfferedCurrentController // optional
	*AdjustablePhaseModeController      // optional
	*AdjustableCableLockController      // optional
}

func (m *MockedChargepoint) AssertExpectations(t mock.TestingT) bool {
	if !m.Controller.AssertExpectations(t) {
		return false
	}

	if m.AdjustableMaxCurrentController != nil && !m.AdjustableMaxCurrentController.AssertExpectations(t) {
		return false
	}

	if m.AdjustableOfferedCurrentController != nil && !m.AdjustableOfferedCurrentController.AssertExpectations(t) {
		return false
	}

	if m.AdjustablePhaseModeController != nil && !m.AdjustablePhaseModeController.AssertExpectations(t) {
		return false
	}

	if m.AdjustableCableLockController != nil && !m.AdjustableCableLockController.AssertExpectations(t) {
		return false
	}

	return true
}

func NewMockedChargepoint(
	controller *Controller,
	maxCurrentController *AdjustableMaxCurrentController,
	offeredCurrentController *AdjustableOfferedCurrentController,
	phaseModeController *AdjustablePhaseModeController,
	cableLockController *AdjustableCableLockController,
) *MockedChargepoint {
	return &MockedChargepoint{
		Controller:                         controller,
		AdjustableMaxCurrentController:     maxCurrentController,
		AdjustableOfferedCurrentController: offeredCurrentController,
		AdjustablePhaseModeController:      phaseModeController,
		AdjustableCableLockController:      cableLockController,
	}
}
