package mockednumericsensor

func MockReporter() *Reporter {
	return &Reporter{}
}

func (_m *Reporter) MockNumericSensorReport(unit string, value float64, err error, once bool) *Reporter {
	c := _m.On("NumericSensorReport", unit).Return(value, err)

	if once {
		c.Once()
	}

	return _m
}
