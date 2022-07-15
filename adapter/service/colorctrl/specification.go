package colorctrl

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
)

// Specification creates a service specification.
func Specification(
	resourceName,
	resourceAddress,
	address string,
	groups,
	supportedComponents []string,
	supportedDurations map[string]int64,
) *fimptype.Service {
	s := &fimptype.Service{
		Address: fmt.Sprintf("/rt:dev/rn:%s/ad:%s/sv:%s/ad:%s", resourceName, resourceAddress, Colorctrl, address),
		Name:    Colorctrl,
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
	}
}
