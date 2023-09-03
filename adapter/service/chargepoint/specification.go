package chargepoint

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

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
func WithSupportedMaxCurrent(current int64) adapter.SpecificationOption {
	return adapter.SpecificationOptionFn(func(f *fimptype.Service) {
		f.Props[PropertySupportedMaxCurrent] = current
	})
}

// WithGridType adds grid type to the service specification.
func WithGridType(gridType string) adapter.SpecificationOption {
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
	groups,
	supportedStates []string,
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
func requiredInterfaces() []fimptype.Interface { //nolint:funlen
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdChargeStart,
			ValueType: fimpgo.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdChargeStop,
			ValueType: fimpgo.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdStateGetReport,
			ValueType: fimpgo.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtStateReport,
			ValueType: fimpgo.VTypeString,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdCableLockSet,
			ValueType: fimpgo.VTypeBool,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdCableLockGetReport,
			ValueType: fimpgo.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtCableLockReport,
			ValueType: fimpgo.VTypeBool,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdCurrentSessionGetReport,
			ValueType: fimpgo.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtCurrentSessionReport,
			ValueType: fimpgo.VTypeFloat,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   router.EvtErrorReport,
			ValueType: fimpgo.VTypeString,
			Version:   "1",
		},
	}
}

// adjustableCurrentInterfaces returns interfaces for adjustable current controller.
func adjustableCurrentInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdCurrentSessionSetCurrent,
			ValueType: fimpgo.VTypeInt,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdMaxCurrentSet,
			ValueType: fimpgo.VTypeInt,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdMaxCurrentGetReport,
			ValueType: fimpgo.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtMaxCurrentReport,
			ValueType: fimpgo.VTypeInt,
			Version:   "1",
		},
	}
}
