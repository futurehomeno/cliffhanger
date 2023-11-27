package mockedoutlvlswitch

import "github.com/stretchr/testify/mock"

type MockedOutSwitchLvl struct {
	*Controller
	*LevelTransitionController // optional
}

func (m *MockedOutSwitchLvl) AssertExpectations(t mock.TestingT) bool {
	if !m.Controller.AssertExpectations(t) {
		return false
	}

	if m.LevelTransitionController != nil && !m.LevelTransitionController.AssertExpectations(t) {
		return false
	}

	return true
}

func NewMockedOutSwitchLvl(
	controller *Controller,
	transitionController *LevelTransitionController,
) *MockedOutSwitchLvl {
	return &MockedOutSwitchLvl{
		Controller:                controller,
		LevelTransitionController: transitionController,
	}
}
