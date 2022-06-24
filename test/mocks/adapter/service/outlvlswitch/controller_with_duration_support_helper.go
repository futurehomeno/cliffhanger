package mockedoutlvlswitch

import time "time"

func (_m *ControllerWithDurationSupport) MockLevelSwitchLevelReport(value int64, err error, once bool) *ControllerWithDurationSupport {
	c := _m.On("LevelSwitchLevelReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *ControllerWithDurationSupport) MockSetLevelSwitchLevel(value int64, err error, once bool) *ControllerWithDurationSupport {
	c := _m.On("SetLevelSwitchLevel", value).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *ControllerWithDurationSupport) MockSetLevelSwitchLevelWithDuration(value int64, duration time.Duration, err error, once bool) *ControllerWithDurationSupport {
	c := _m.On("SetLevelSwitchLevelWithDuration", value, duration).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *ControllerWithDurationSupport) MockSetLevelSwitchBinaryState(value bool, err error, once bool) *ControllerWithDurationSupport {
	c := _m.On("SetLevelSwitchBinaryState", value).Return(err)

	if once {
		c.Once()
	}

	return _m
}
