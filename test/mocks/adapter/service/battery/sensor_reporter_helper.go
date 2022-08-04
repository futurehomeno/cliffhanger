package mockedbattery

import battery "github.com/futurehomeno/cliffhanger/adapter/service/battery"

func (_m *SensorReporter) MockBatteryAlarmReport(alarm battery.AlarmReport, err error, once bool) *SensorReporter {
	c := _m.On("BatteryAlarmReport").Return(alarm, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *SensorReporter) MockBatteryFullReport(full battery.FullReport, err error, once bool) *SensorReporter {
	c := _m.On("BatteryFullReport").Return(full, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *SensorReporter) MockBatterySensorReport(sensor float64, unit string, err error, once bool) *SensorReporter {
	c := _m.On("BatterySensorReport").Return(sensor, unit, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *SensorReporter) MockBatteryLevelReport(level int64, state string, err error, once bool) *SensorReporter {
	c := _m.On("BatteryLevelReport").Return(level, state, err)

	if once {
		c.Once()
	}

	return _m
}
