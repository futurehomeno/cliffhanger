package mockedchargepoint

import (
	"github.com/stretchr/testify/mock"
)

type MockedChargepoint struct {
	*Controller
	*AdjustableCurrentController
}

func (m *MockedChargepoint) AssertExpectations(t mock.TestingT) bool {
	if !m.Controller.AssertExpectations(t) {
		return false
	}

	return m.AdjustableCurrentController.AssertExpectations(t)
}

func NewMockedChargepoint(controller *Controller, adjustableCurrentController *AdjustableCurrentController) *MockedChargepoint {
	return &MockedChargepoint{
		Controller:                  controller,
		AdjustableCurrentController: adjustableCurrentController,
	}
}
