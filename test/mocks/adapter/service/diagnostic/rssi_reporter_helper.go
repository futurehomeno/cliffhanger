package mockeddiagnostic

func (_m *RSSIReporter) MockRSSIReport(value int, err error, once bool) *RSSIReporter {
	c := _m.On("RSSIReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}
