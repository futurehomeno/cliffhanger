package colorctrl

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/router"
)

// Specification creates a service specification.
func Specification(
	resourceName,
	resourceAddress,
	address string,
	groups,
	supportedComponents []string,
	supportedDurations map[string]int,
) *fimptype.Service {
	s := &fimptype.Service{
		Address: fmt.Sprintf("/rt:dev/rn:%s/ad:%s/sv:%s/ad:%s", resourceName, resourceAddress, ColorCtrl, address),
		Name:    ColorCtrl,
		Groups:  groups,
		Enabled: true,
		Props: map[string]interface{}{
			PropertySupportedComponents: supportedComponents,
		},
		Interfaces: requiredInterfaces(),
	}

	if len(supportedDurations) != 0 {
		s.Props[PropertySupportedDurations] = supportedDurations
	}

	return s
}

// requiredInterfaces returns required interfaces by the service.
func requiredInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdColorSet,
			ValueType: fimpgo.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtColorReport,
			ValueType: fimpgo.VTypeIntMap,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdColorGetReport,
			ValueType: fimpgo.VTypeNull,
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
