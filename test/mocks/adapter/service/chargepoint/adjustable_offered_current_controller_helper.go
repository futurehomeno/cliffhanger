package mockedchargepoint

func (_m *AdjustableOfferedCurrentController) MockSetChargepointOfferedCurrent(value int64, err error, once bool) *AdjustableOfferedCurrentController {
	c := _m.On("SetChargepointOfferedCurrent", value).Return(err)

	if once {
		c.Once()
	}

	return _m
}
