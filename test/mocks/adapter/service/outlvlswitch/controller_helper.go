package mockedoutlvlswitch

func (_m *Controller) MockLevelSwitchLevelReport(value int64, err error, once bool) *Controller {
	c := _m.On("LevelSwitchLevelReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockSetLevelSwitchLevel(value int64, err error, once bool) *Controller {
	c := _m.On("SetLevelSwitchLevel", value).Return(err)

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
