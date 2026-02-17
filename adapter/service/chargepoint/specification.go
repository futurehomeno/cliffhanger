package chargepoint

import (
	"fmt"

	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/types"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// WithChargingModes adds supported charging modes to the service specification.
func WithChargingModes(modes ...string) adapter.SpecificationOption {
	return adapter.SpecificationOptionFn(func(f *fimptype.Service) {
		f.Props[PropertySupportedChargingModes] = modes
	})
}

// WithSupportedMaxCurrent adds supported max current to the service specification.
func WithSupportedMaxCurrent(current int) adapter.SpecificationOption {
	return adapter.SpecificationOptionFn(func(f *fimptype.Service) {
		f.Props[PropertySupportedMaxCurrent] = current
	})
}

// WithSupportedPhaseModes adds phases to the service specification.
func WithSupportedPhaseModes(modes ...types.PhaseMode) adapter.SpecificationOption {
	return adapter.SpecificationOptionFn(func(f *fimptype.Service) {
		f.Props[PropertySupportedPhaseModes] = modes
	})
}

// WithGridType adds grid type to the service specification.
func WithGridType(gridType types.GridType) adapter.SpecificationOption {
	return adapter.SpecificationOptionFn(func(f *fimptype.Service) {
		f.Props[PropertyGridType] = gridType
	})
}

// WithPhases adds phases to the service specification.
func WithPhases(phases int) adapter.SpecificationOption {
	return adapter.SpecificationOptionFn(func(f *fimptype.Service) {
		f.Props[PropertyPhases] = phases
	})
}

// Specification creates a service specification.
func Specification(
	resourceName,
	resourceAddress,
	address string,
	groups []string,
	supportedStates []State,
	options ...adapter.SpecificationOption,
) *fimptype.Service {
	s := &fimptype.Service{
		Address: fmt.Sprintf("/rt:dev/rn:%s/ad:%s/sv:%s/ad:%s", resourceName, resourceAddress, Chargepoint, address),
		Name:    Chargepoint,
		Groups:  groups,
		Enabled: true,
		Props: map[string]interface{}{
			PropertySupportedStates: supportedStates,
		},
		Interfaces: requiredInterfaces(),
	}

	for _, o := range options {
		o.Apply(s)
	}

	return s
}

// requiredInterfaces returns required interfaces by the service.
func requiredInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdChargeStart,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdChargeStop,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdStateGetReport,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtStateReport,
			ValueType: fimptype.VTypeString,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdCurrentSessionGetReport,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtCurrentSessionReport,
			ValueType: fimptype.VTypeFloat,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   router.EvtErrorReport,
			ValueType: fimptype.VTypeString,
			Version:   "1",
		},
	}
}

// adjustableCableLockInterfaces returns interfaces for adjustable cable lock controller.
func adjustableCableLockInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdCableLockSet,
			ValueType: fimptype.VTypeBool,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdCableLockGetReport,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtCableLockReport,
			ValueType: fimptype.VTypeBool,
			Version:   "1",
		},
	}
}

// cableLockAwareInterfaces returns interfaces for cable lock aware controller.
func cableLockAwareInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdCableLockGetReport,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtCableLockReport,
			ValueType: fimptype.VTypeBool,
			Version:   "1",
		},
	}
}

// adjustableMaxCurrentInterfaces returns interfaces for adjustable max current controller.
func adjustableMaxCurrentInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdMaxCurrentSet,
			ValueType: fimptype.VTypeInt,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdMaxCurrentGetReport,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtMaxCurrentReport,
			ValueType: fimptype.VTypeInt,
			Version:   "1",
		},
	}
}

// adjustableCurrentInterfaces returns interfaces for adjustable current controller.
func adjustablePhaseModeInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdPhaseModeSet,
			ValueType: fimptype.VTypeString,
			Version:   "1",
		},
	}
}

// phaseModeAwareInterfaces returns interfaces for phase mode aware controller.
func phaseModeAwareInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdPhaseModeGetReport,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtPhaseModeReport,
			ValueType: fimptype.VTypeString,
			Version:   "1",
		},
	}
}

// adjustableOfferedCurrentInterfaces returns interfaces for adjustable offered current controller.
func adjustableOfferedCurrentInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdCurrentSessionSetCurrent,
			ValueType: fimptype.VTypeInt,
			Version:   "1",
		},
	}
}
