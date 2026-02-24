package fanctrl

import (
	"fmt"

	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/router"
)

// Specification creates a service specification.
func Specification(
	resourceName,
	resourceAddress,
	address string,
	groups,
	supportedModes []string,
) *fimptype.Service {
	s := &fimptype.Service{
		Address: fmt.Sprintf("/rt:dev/rn:%s/ad:%s/sv:%s/ad:%s", resourceName, resourceAddress, FanCtrl, address),
		Name:    FanCtrl,
		Groups:  groups,
		Enabled: true,
		Props: map[string]any{
			PropertySupportedModes: supportedModes,
		},
		Interfaces: requiredInterfaces(),
	}

	return s
}

// requiredInterfaces returns required interfaces by the service.
func requiredInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdModeSet,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtModeReport,
			ValueType: fimptype.VTypeIntMap,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdModeGetReport,
			ValueType: fimptype.VTypeNull,
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
