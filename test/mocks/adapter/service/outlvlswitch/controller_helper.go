package mockedoutlvlswitch

func (_m *Controller) MockLevelReport(value int64, err error, once bool) *Controller {
	c := _m.On("LevelReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockBinaryReport(value bool, err error, once bool) *Controller {
	c := _m.On("BinaryReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockSetLevelCtrl(value int64, err error, once bool) *Controller {
	c := _m.On("SetLevelCtrl", value).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockSetLevelWithDurationCtrl(value int64, duration int64, err error, once bool) *Controller {
	c := _m.On("SetLevelWithDurationCtrl", value, duration).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockSetBinaryCtrl(value bool, err error, once bool) *Controller {
	c := _m.On("SetBinaryCtrl", value).Return(err)

	if once {
		c.Once()
	}

	return _m
}
