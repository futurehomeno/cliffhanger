package mockednumericmeter

import (
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
)

func (_m *Reporter) MockMeterReport(unit numericmeter.Unit, value float64, err error, once bool) *Reporter {
	c := _m.On("MeterReport", unit).Return(value, err)

	if once {
		c.Once()
	}

	return _m
}
