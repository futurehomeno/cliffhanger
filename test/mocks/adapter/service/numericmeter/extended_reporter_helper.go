package mockednumericmeter

func (_m *ExtendedReporter) MockMeterExtendedReport(values string, value map[string]float64, err error, once bool) *ExtendedReporter {
	c := _m.On("MeterExtendedReport", values).Return(value, err)

	if once {
		c.Once()
	}

	return _m
}
