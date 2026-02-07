package mockedoutlvlswitch

import (
	"github.com/futurehomeno/cliffhanger/adapter/service/outlvlswitch"
)

func (_m *LevelTransitionController) MockStartLevelTransition(value string, params outlvlswitch.LevelTransitionParams, err error) *LevelTransitionController {
	_m.On("StartLevelTransition", value, params).Return(err)

	return _m
}

func (_m *LevelTransitionController) MockStopLevelTransition(err error) *LevelTransitionController {
	_m.On("StopLevelTransition").Return(err)

	return _m
}
