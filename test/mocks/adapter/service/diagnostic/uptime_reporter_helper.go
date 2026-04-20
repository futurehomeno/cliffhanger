package mockeddiagnostic

func (_m *UptimeReporter) MockUptimeReport(value int, err error, once bool) *UptimeReporter {
	c := _m.On("UptimeReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}
