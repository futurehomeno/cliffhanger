package mockedchargepoint

import (
	"github.com/futurehomeno/cliffhanger/types"
)

func (_m *AdjustablePhaseModeController) MockChargepointPhaseModeReport(value types.PhaseMode, err error, once bool) *AdjustablePhaseModeController {
	c := _m.On("ChargepointPhaseModeReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *AdjustablePhaseModeController) MockSetChargepointPhaseMode(value types.PhaseMode, err error, once bool) *AdjustablePhaseModeController {
	c := _m.On("SetChargepointPhaseMode", value).Return(err)

	if once {
		c.Once()
	}

	return _m
}
