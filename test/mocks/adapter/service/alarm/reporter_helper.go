package mockedalarm

import "github.com/futurehomeno/cliffhanger/adapter/service/alarm"

func (_m *Reporter) MockAlarmReport(report *alarm.Report, event string, err error, once bool) *Reporter {
	c := _m.On("AlarmReport", event).Return(report, err)

	if once {
		c.Once()
	}

	return _m
}
