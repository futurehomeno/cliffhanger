package thermostat

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter/service/waterheater"
)

func Specification(
	resourceName,
	resourceAddress,
	address string,
	groups,
	supportedModes,
	supportedSetpoints,
	supportedStates []string,
) *fimptype.Service {
	return &fimptype.Service{
		Address: fmt.Sprintf("/rt:dev/rn:%s/ad:%s/sv:%s/ad:%s", resourceName, resourceAddress, Thermostat, address),
		Name:    Thermostat,
		Groups:  groups,
		Enabled: true,
		Props: map[string]interface{}{
			waterheater.PropertySupportedModes:     supportedModes,
			waterheater.PropertySupportedSetpoints: supportedSetpoints,
			waterheater.PropertySupportedStates:    supportedStates,
		},
		Interfaces: requiredInterfaces(),
	}
}

func requiredInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      "in",
			MsgType:   CmdModeGetReport,
			ValueType: fimpgo.VTypeNull,
			Version:   "1",
		},
		{
			Type:      "in",
			MsgType:   CmdModeSet,
			ValueType: fimpgo.VTypeString,
			Version:   "1",
		},
		{
			Type:      "out",
			MsgType:   EvtModeReport,
			ValueType: fimpgo.VTypeString,
			Version:   "1",
		},
		{
			Type:      "in",
			MsgType:   CmdSetpointGetReport,
			ValueType: fimpgo.VTypeString,
			Version:   "1",
		},
		{
			Type:      "in",
			MsgType:   CmdSetpointSet,
			ValueType: fimpgo.VTypeStrMap,
			Version:   "1",
		},
		{
			Type:      "out",
			MsgType:   EvtSetpointReport,
			ValueType: fimpgo.VTypeStrMap,
			Version:   "1",
		},
		{
			Type:      "in",
			MsgType:   CmdStateGetReport,
			ValueType: fimpgo.VTypeNull,
			Version:   "1",
		},
		{
			Type:      "out",
			MsgType:   EvtStateReport,
			ValueType: fimpgo.VTypeString,
			Version:   "1",
		},
	}
}
