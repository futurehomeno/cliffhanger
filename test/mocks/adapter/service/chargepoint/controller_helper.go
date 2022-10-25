package mockedchargepoint

import (
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
)

func (_m *Controller) MockStartChargepointCharging(mode string, err error, once bool) *Controller {
	c := _m.On("StartChargepointCharging", mode).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockStopChargepointCharging(err error, once bool) *Controller {
	c := _m.On("StopChargepointCharging").Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockChargepointStateReport(report chargepoint.StateReport, err error, once bool) *Controller {
	c := _m.On("ChargepointStateReport").Return(report, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockChargepointCurrentSessionReport(value float64, err error, once bool) *Controller {
	c := _m.On("ChargepointCurrentSessionReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockSetChargepointCableLock(lock bool, err error, once bool) *Controller {
	c := _m.On("SetChargepointCableLock", lock).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockChargepointCableLockReport(lock bool, err error, once bool) *Controller {
	c := _m.On("ChargepointCableLockReport").Return(lock, err)

	if once {
		c.Once()
	}

	return _m
}
