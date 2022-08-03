package mockedbattery

import battery "github.com/futurehomeno/cliffhanger/adapter/service/battery"

func (_m *Reporter) MockBatteryAlarmReport(alarm battery.AlarmReport, err error, once bool) *Reporter {
	c := _m.On("BatteryAlarmReport").Return(alarm, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Reporter) MockBatteryFullReport(full battery.FullReport, err error, once bool) *Reporter {
	c := _m.On("BatteryFullReport").Return(full, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Reporter) MockBatteryHealthReport(health int64, err error, once bool) *Reporter {
	c := _m.On("BatteryHealthReport").Return(health, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Reporter) MockBatteryLevelReport(level int64, state string, err error, once bool) *Reporter {
	c := _m.On("BatteryLevelReport").Return(level, state, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Reporter) MockBatterySensorReport(sensor float64, unit string, err error, once bool) *Reporter {
	c := _m.On("BatterySensorReport").Return(sensor, unit, err)

	if once {
		c.Once()
	}

	return _m
}
