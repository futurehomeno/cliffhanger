package outlvlswitch

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
)

// Specificatiion creates a service specification.
func Specification(
	resourceName,
	resourceAddress,
	address,
	maxLvl,
	minLvl,
	switchType string,
	groups []string,
) *fimptype.Service {
	return &fimptype.Service{
		Address: fmt.Sprintf("/rt:dev/rn:%s/ad:%s/sv:%s/ad:%s", resourceName, resourceAddress, OutLvlSwitch, address),
		Name:    OutLvlSwitch,
		Groups:  groups,
		Enabled: true,
		Props: map[string]interface{}{
			MaxLvl:     maxLvl,
			MinLvl:     minLvl,
			SwitchType: switchType,
		},
		Interfaces: requiredInterfaces(),
	}
}

// requiredInterfaces returns required interfaces by the service.
func requiredInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdLvlSet,
			ValueType: fimpgo.VTypeInt,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdLvlStart,
			ValueType: fimpgo.VTypeString,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdLvlStop,
			ValueType: fimpgo.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdBinarySet,
			ValueType: fimpgo.VTypeBool,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdLvlGetReport,
			ValueType: fimpgo.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtLvlReport,
			ValueType: fimpgo.VTypeInt,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtBinaryReport,
			ValueType: fimpgo.VTypeBool,
			Version:   "1",
		},
	}
}
