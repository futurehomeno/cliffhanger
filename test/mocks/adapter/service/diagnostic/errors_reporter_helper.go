package mockeddiagnostic

func (_m *ErrorsReporter) MockErrorsReport(value []string, err error, once bool) *ErrorsReporter {
	c := _m.On("ErrorsReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}
