package mockedchargepoint

func MockEnergyAwareController() *EnergyAwareController {
	return &EnergyAwareController{}
}

func (_m *EnergyAwareController) MockChargepointChargingModeReport(report string, err error, once bool) *EnergyAwareController {
	c := _m.On("ChargepointChargingModeReport").Return(report, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *EnergyAwareController) MockSetChargepointChargingMode(mode string, err error, once bool) *EnergyAwareController {
	c := _m.On("SetChargepointChargingMode", mode).Return(err)

	if once {
		c.Once()
	}

	return _m
}
