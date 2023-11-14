package mockedoutlvlswitch

import "time"

func (_m *LevelTransitionController) MockStartLevelTransition(value string, startLevel int, duration time.Duration, err error) *LevelTransitionController {
	_m.On("StartLevelTransition", value, startLevel, duration).Return(err)

	return _m
}

func (_m *LevelTransitionController) MockStopLevelTransition(err error) *LevelTransitionController {
	_m.On("StopLevelTransition").Return(err)

	return _m
}
