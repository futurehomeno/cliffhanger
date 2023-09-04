package mockednumericmeter

import (
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
)

func (_m *ExtendedReporter) MockMeterExtendedReport(values string, value numericmeter.ValuesReport, err error, once bool) *ExtendedReporter {
	c := _m.On("MeterExtendedReport", values).Return(value, err)

	if once {
		c.Once()
	}

	return _m
}
