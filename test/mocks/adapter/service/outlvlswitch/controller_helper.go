package mockedoutlvlswitch

import (
	time "time"
)

func (_m *Controller) MockLevelSwitchLevelReport(value int, err error, once bool) *Controller {
	c := _m.On("LevelSwitchLevelReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockLevelSwitchBinaryStateReport(value bool, err error, once bool) *Controller {
	c := _m.On("LevelSwitchBinaryStateReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockSetLevelSwitchLevel(value int, duration time.Duration, err error, once bool) *Controller {
	c := _m.On("SetLevelSwitchLevel", value, duration).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockSetLevelSwitchBinaryState(value bool, err error, once bool) *Controller {
	c := _m.On("SetLevelSwitchBinaryState", value).Return(err)

	if once {
		c.Once()
	}

	return _m
}
