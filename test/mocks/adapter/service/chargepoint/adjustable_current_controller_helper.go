package mockedchargepoint

func (_m *AdjustableCurrentController) MockChargepointMaxCurrentReport(value int64, err error, once bool) *AdjustableCurrentController {
	c := _m.On("ChargepointMaxCurrentReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *AdjustableCurrentController) MockSetChargepointMaxCurrent(value int64, err error, once bool) *AdjustableCurrentController {
	c := _m.On("SetChargepointMaxCurrent", value).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *AdjustableCurrentController) MockSetChargepointOfferedCurrent(value int64, err error, once bool) *AdjustableCurrentController {
	c := _m.On("SetChargepointOfferedCurrent", value).Return(err)

	if once {
		c.Once()
	}

	return _m
}
