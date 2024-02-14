package mockedfanctrl

// MockGetMode creates a controller for testing.
func (_m *Controller) MockGetMode(mode string, err error, once bool) *Controller {
	c := _m.On("FanCtrlModeReport").Return(mode, err)

	if once {
		c.Once()
	}

	return _m
}

// MockSetMode creates a controller for testing.
func (_m *Controller) MockSetMode(mode string, err error, once bool) *Controller {
	c := _m.On("SetFanCtrlMode", mode).Return(err)

	if once {
		c.Once()
	}

	return _m
}
