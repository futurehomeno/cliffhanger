package outlvlswitch

import (
	"fmt"

	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// WithSupportedDuration updates specification to allow support of the duration property.
func WithSupportedDuration() adapter.SpecificationOption {
	return adapter.SpecificationOptionFn(func(f *fimptype.Service) {
		f.Props[PropertySupportDuration] = true
	})
}

// WithSupportedStartLevel updates specification to allow support of the start level property.
func WithSupportedStartLevel() adapter.SpecificationOption {
	return adapter.SpecificationOptionFn(func(f *fimptype.Service) {
		f.Props[PropertySupportStartLevel] = true
	})
}

// Specification creates a service specification.
func Specification(
	resourceName,
	resourceAddress,
	address,
	switchType string,
	maxLvl,
	minLvl int,
	groups []string,
	options ...adapter.SpecificationOption,
) *fimptype.Service {
	s := &fimptype.Service{
		Address: fmt.Sprintf("/rt:dev/rn:%s/ad:%s/sv:%s/ad:%s", resourceName, resourceAddress, OutLvlSwitch, address),
		Name:    OutLvlSwitch,
		Groups:  groups,
		Enabled: true,
		Props: map[string]any{
			PropertyMaxLvl:     maxLvl,
			PropertyMinLvl:     minLvl,
			PropertySwitchType: switchType,
		},
		Interfaces: requiredInterfaces(),
	}

	for _, op := range options {
		op.Apply(s)
	}

	return s
}

// requiredInterfaces returns required interfaces by the service.
func requiredInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdLvlSet,
			ValueType: fimptype.VTypeInt,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdBinarySet,
			ValueType: fimptype.VTypeBool,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdLvlGetReport,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtLvlReport,
			ValueType: fimptype.VTypeInt,
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

func levelTransitionInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdLvlStart,
			ValueType: fimptype.VTypeString,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdLvlStop,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
	}
}
