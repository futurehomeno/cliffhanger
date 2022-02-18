package waterheater

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/router"
)

// Range represents setpoint acceptable range.
type Range struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// Specification creates a service specification.
// Supported range and supported ranges are optional parameters.
func Specification(
	resourceName,
	resourceAddress,
	address string,
	groups,
	supportedModes,
	supportedSetpoints,
	supportedStates []string,
	supportedRange *Range,
	supportedRanges map[string]Range,
	supportedStep float64,
) *fimptype.Service {
	s := &fimptype.Service{
		Address: fmt.Sprintf("/rt:dev/rn:%s/ad:%s/sv:%s/ad:%s", resourceName, resourceAddress, WaterHeater, address),
		Name:    WaterHeater,
		Groups:  groups,
		Enabled: true,
		Props: map[string]interface{}{
			PropertySupportedModes:     supportedModes,
			PropertySupportedSetpoints: supportedSetpoints,
			PropertySupportedStates:    supportedStates,
			PropertySupportedStep:      supportedStep,
		},
		Interfaces: requiredInterfaces(),
	}

	if supportedRanges != nil {
		s.Props[PropertySupportedRanges] = supportedRanges
	} else if supportedRange != nil {
		s.Props[PropertySupportedRange] = *supportedRange
	}

	return s
}

// requiredInterfaces returns required interfaces by the service.
func requiredInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdModeGetReport,
			ValueType: fimpgo.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdModeSet,
			ValueType: fimpgo.VTypeString,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtModeReport,
			ValueType: fimpgo.VTypeString,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdSetpointGetReport,
			ValueType: fimpgo.VTypeString,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdSetpointSet,
			ValueType: fimpgo.VTypeStrMap,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtSetpointReport,
			ValueType: fimpgo.VTypeStrMap,
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
			Type:      fimptype.TypeOut,
			MsgType:   router.EvtErrorReport,
			ValueType: fimpgo.VTypeString,
			Version:   "1",
		},
	}
}
