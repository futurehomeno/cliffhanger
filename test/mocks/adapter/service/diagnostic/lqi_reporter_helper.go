package mockeddiagnostic

func (_m *LQIReporter) MockLQIReport(value int, err error, once bool) *LQIReporter {
	c := _m.On("LQIReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}
