package mockednumericmeter

func (_m *Reporter) MockMeterReport(unit string, value float64, err error, once bool) *Reporter {
	c := _m.On("MeterReport", unit).Return(value, err)

	if once {
		c.Once()
	}

	return _m
}
