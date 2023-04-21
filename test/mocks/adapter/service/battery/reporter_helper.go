package mockedbattery

import "github.com/futurehomeno/cliffhanger/adapter/service/battery"

func (_m *Reporter) MockBatteryAlarmReport(alarm *battery.AlarmReport, event string, err error, once bool) *Reporter {
	c := _m.On("BatteryAlarmReport", event).Return(alarm, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Reporter) MockBatteryLevelReport(level int64, err error, once bool) *Reporter {
	c := _m.On("BatteryLevelReport").Return(level, err)

	if once {
		c.Once()
	}

	return _m
}
