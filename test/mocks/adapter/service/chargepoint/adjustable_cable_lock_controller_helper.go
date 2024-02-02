package mockedchargepoint

import "github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"

func (_m *AdjustableCableLockController) MockSetChargepointCableLock(lock bool, err error, once bool) *AdjustableCableLockController {
	c := _m.On("SetChargepointCableLock", lock).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *AdjustableCableLockController) MockChargepointCableLockReport(cableReport *chargepoint.CableReport, err error, once bool) *AdjustableCableLockController {
	c := _m.On("ChargepointCableLockReport").Return(cableReport, err)

	if once {
		c.Once()
	}

	return _m
}
