package mockedbattery

func (_m *Reporter) MockBatteryAlarmReport(alarm map[string]string, err error, once bool) *Reporter {
	c := _m.On("BatteryAlarmReport").Return(alarm, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Reporter) MockBatteryFullReport(full interface{}, err error, once bool) *Reporter {
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

func (_m *Reporter) MockBatteryLevelReport(level int64, err error, once bool) *Reporter {
	c := _m.On("BatteryLevelReport").Return(level, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Reporter) MockBatterySensorReport(sensor float64, err error, once bool) *Reporter {
	c := _m.On("BatterySensorReport").Return(sensor, err)

	if once {
		c.Once()
	}

	return _m
}
