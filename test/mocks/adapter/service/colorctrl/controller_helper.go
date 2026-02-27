package mockedcolorctrl

func (_m *Controller) MockSetColorCtrlColor(color map[string]int, err error, once bool) *Controller {
	c := _m.On("SetColorCtrlColor", color).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockColorCtrlColorReport(color map[string]int, err error, once bool) *Controller {
	c := _m.On("ColorCtrlColorReport").Return(color, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockStartColorCtrlTransition(transitionObject map[string]any, err error, once bool) *Controller {
	c := _m.On("StartColorCtrlTransition", transitionObject).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockStopColorCtrlTransition(value string, err error, once bool) *Controller {
	c := _m.On("StopColorCtrlTransition", value).Return(err)

	if once {
		c.Once()
	}

	return _m
}
