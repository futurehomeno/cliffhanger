package mockedchargepoint

func (_m *AdjustableMaxCurrentController) MockChargepointMaxCurrentReport(value int64, err error, once bool) *AdjustableMaxCurrentController {
	c := _m.On("ChargepointMaxCurrentReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *AdjustableMaxCurrentController) MockSetChargepointMaxCurrent(value int64, err error, once bool) *AdjustableMaxCurrentController {
	c := _m.On("SetChargepointMaxCurrent", value).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *AdjustableOfferedCurrentController) MockSetChargepointOfferedCurrent(value int64, err error, once bool) *AdjustableOfferedCurrentController {
	c := _m.On("SetChargepointOfferedCurrent", value).Return(err)

	if once {
		c.Once()
	}

	return _m
}
