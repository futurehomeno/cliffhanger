package mockedwaterheater

func (_m *Controller) MockSetWaterHeaterMode(mode string, err error, once bool) *Controller {
	c := _m.On("SetWaterHeaterMode", mode).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockSetWaterHeaterSetpoint(mode string, value float64, unit string, err error, once bool) *Controller {
	c := _m.On("SetWaterHeaterSetpoint", mode, value, unit).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockWaterHeaterModeReport(mode string, err error, once bool) *Controller {
	c := _m.On("WaterHeaterModeReport").Return(mode, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockWaterHeaterSetpointReport(mode string, value float64, unit string, err error, once bool) *Controller {
	c := _m.On("WaterHeaterSetpointReport", mode).Return(value, unit, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockWaterHeaterStateReport(state string, err error, once bool) *Controller {
	c := _m.On("WaterHeaterStateReport").Return(state, err)

	if once {
		c.Once()
	}

	return _m
}
