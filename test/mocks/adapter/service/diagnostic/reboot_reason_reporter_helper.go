package mockeddiagnostic

func (_m *RebootReasonReporter) MockRebootReasonReport(value string, err error, once bool) *RebootReasonReporter {
	c := _m.On("RebootReasonReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}
