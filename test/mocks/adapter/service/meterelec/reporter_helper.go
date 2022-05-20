package mockedmeterelec

func (_m *Reporter) MockElectricityMeterReport(unit string, value float64, err error, once bool) *Reporter {
	c := _m.On("ElectricityMeterReport", unit).Return(value, err)

	if once {
		c.Once()
	}

	return _m
}
