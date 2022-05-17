package mockedthermostat

func MockController() *Controller {
	return &Controller{}
}

func (_m *Controller) MockSetThermostatMode(mode string, err error, once bool) *Controller {
	c := _m.On("SetThermostatMode", mode).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockSetThermostatSetpoint(mode string, value float64, unit string, err error, once bool) *Controller {
	c := _m.On("SetThermostatSetpoint", mode, value, unit).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockThermostatModeReport(mode string, err error, once bool) *Controller {
	c := _m.On("ThermostatModeReport").Return(mode, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockThermostatSetpointReport(mode string, value float64, unit string, err error, once bool) *Controller {
	c := _m.On("ThermostatSetpointReport", mode).Return(value, unit, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockThermostatStateReport(state string, err error, once bool) *Controller {
	c := _m.On("ThermostatStateReport").Return(state, err)

	if once {
		c.Once()
	}

	return _m
}


