package mockedoutbinswitch

func (_m *Controller) MockedBinarySwitchBinarySet(value bool, err error, once bool) *Controller {
	c := _m.On("SetBinarySwitchState", value).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockedBinarySwitchBinaryReport(value any, err error, once bool) *Controller {
	c := _m.On("BinarySwitchStateReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}
