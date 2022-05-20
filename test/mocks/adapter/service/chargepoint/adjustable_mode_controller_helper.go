package mockedchargepoint

func (_m *AdjustableModeController) MockChargepointChargingModeReport(report string, err error, once bool) *AdjustableModeController {
	c := _m.On("ChargepointChargingModeReport").Return(report, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *AdjustableModeController) MockSetChargepointChargingMode(mode string, err error, once bool) *AdjustableModeController {
	c := _m.On("SetChargepointChargingMode", mode).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *AdjustableModeController) MockStartChargepointCharging(err error, once bool) *AdjustableModeController {
	c := _m.On("StartChargepointCharging").Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *AdjustableModeController) MockStopChargepointCharging(err error, once bool) *AdjustableModeController {
	c := _m.On("StopChargepointCharging").Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *AdjustableModeController) MockChargepointStateReport(state string, err error, once bool) *AdjustableModeController {
	c := _m.On("ChargepointStateReport").Return(state, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *AdjustableModeController) MockChargepointCurrentSessionReport(value float64, err error, once bool) *AdjustableModeController {
	c := _m.On("ChargepointCurrentSessionReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *AdjustableModeController) MockSetChargepointCableLock(lock bool, err error, once bool) *AdjustableModeController {
	c := _m.On("SetChargepointCableLock", lock).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *AdjustableModeController) MockChargepointCableLockReport(lock bool, err error, once bool) *AdjustableModeController {
	c := _m.On("ChargepointCableLockReport").Return(lock, err)

	if once {
		c.Once()
	}

	return _m
}
