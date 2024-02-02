package mockedchargepoint

import (
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
)

func (_m *Controller) MockStartChargepointCharging(settings *chargepoint.ChargingSettings, err error, once bool) *Controller {
	c := _m.On("StartChargepointCharging", settings).Return(err)

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

func (_m *Controller) MockChargepointStateReport(state chargepoint.State, err error, once bool) *Controller {
	c := _m.On("ChargepointStateReport").Return(state, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockChargepointCurrentSessionReport(value *chargepoint.SessionReport, err error, once bool) *Controller {
	c := _m.On("ChargepointCurrentSessionReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}
