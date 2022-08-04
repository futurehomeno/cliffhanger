package mockedbattery

import battery "github.com/futurehomeno/cliffhanger/adapter/service/battery"

func (_m *HealthReporter) MockBatteryAlarmReport(alarm battery.AlarmReport, err error, once bool) *HealthReporter {
	c := _m.On("BatteryAlarmReport").Return(alarm, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *HealthReporter) MockBatteryFullReport(full battery.FullReport, err error, once bool) *HealthReporter {
	c := _m.On("BatteryFullReport").Return(full, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *HealthReporter) MockBatteryHealthReport(health int64, err error, once bool) *HealthReporter {
	c := _m.On("BatteryHealthReport").Return(health, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *HealthReporter) MockBatteryLevelReport(level int64, state string, err error, once bool) *HealthReporter {
	c := _m.On("BatteryLevelReport").Return(level, state, err)

	if once {
		c.Once()
	}

	return _m
}
