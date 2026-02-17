package battery

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
	groups []string,
	events []string,
) *fimptype.Service {
	s := &fimptype.Service{
		Address: fmt.Sprintf("/rt:dev/rn:%s/ad:%s/sv:%s/ad:%s", resourceName, resourceAddress, Battery, address),
		Name:    Battery,
		Groups:  groups,
		Enabled: true,
		Props: map[string]interface{}{
			PropertySupportedEvents: events,
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
			MsgType:   CmdLevelGetReport,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtLevelReport,
			ValueType: fimptype.VTypeString,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtAlarmReport,
			ValueType: fimptype.VTypeStrMap,
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
