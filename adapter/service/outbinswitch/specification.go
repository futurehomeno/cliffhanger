package outbinswitch

import (
	"fmt"

	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// Specification creates a service specification.
func Specification(
	resourceName,
	resourceAddress,
	address string,
	groups []string,
	options ...adapter.SpecificationOption,
) *fimptype.Service {
	s := &fimptype.Service{
		Address:    fmt.Sprintf("/rt:dev/rn:%s/ad:%s/sv:%s/ad:%s", resourceName, resourceAddress, OutBinSwitch, address),
		Name:       OutBinSwitch,
		Groups:     groups,
		Enabled:    true,
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
			MsgType:   CmdBinarySet,
			ValueType: fimptype.VTypeBool,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdBinaryGetReport,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtBinaryReport,
			ValueType: fimptype.VTypeBool,
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
