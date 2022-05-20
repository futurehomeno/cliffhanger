package mockedmeterelec

func (_m *ExtendedReporter) MockElectricityMeterReport(unit string, value float64, err error, once bool) *ExtendedReporter {
	c := _m.On("ElectricityMeterReport", unit).Return(value, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *ExtendedReporter) MockElectricityMeterExtendedReport(value map[string]float64, err error, once bool) *ExtendedReporter {
	c := _m.On("ElectricityMeterExtendedReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}
