package mockedchargepoint

import (
	"github.com/stretchr/testify/mock"
)

type MockedChargepoint struct {
	*Controller
	*AdjustableCurrentController   // optional
	*AdjustablePhaseModeController // optional
}

func (m *MockedChargepoint) AssertExpectations(t mock.TestingT) bool {
	if !m.Controller.AssertExpectations(t) {
		return false
	}

	if m.AdjustableCurrentController != nil && !m.AdjustableCurrentController.AssertExpectations(t) {
		return false
	}

	if m.AdjustablePhaseModeController != nil && !m.AdjustablePhaseModeController.AssertExpectations(t) {
		return false
	}

	return true
}

func NewMockedChargepoint(
	controller *Controller,
	currentController *AdjustableCurrentController,
	phaseModeController *AdjustablePhaseModeController,
) *MockedChargepoint {
	return &MockedChargepoint{
		Controller:                    controller,
		AdjustableCurrentController:   currentController,
		AdjustablePhaseModeController: phaseModeController,
	}
}
