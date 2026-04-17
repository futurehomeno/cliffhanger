package mockeddiagnostic

func (_m *RebootsCountReporter) MockRebootsCountReport(value int, err error, once bool) *RebootsCountReporter {
	c := _m.On("RebootsCountReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}
