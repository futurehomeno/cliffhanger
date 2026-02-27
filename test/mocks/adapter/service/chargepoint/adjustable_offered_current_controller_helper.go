package mockedchargepoint

func (_m *AdjustableOfferedCurrentController) MockSetChargepointOfferedCurrent(value int, err error, once bool) *AdjustableOfferedCurrentController {
	c := _m.On("SetChargepointOfferedCurrent", value).Return(err)

	if once {
		c.Once()
	}

	return _m
}
