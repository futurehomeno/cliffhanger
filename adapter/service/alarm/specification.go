package alarm

import (
	"fmt"

	"github.com/futurehomeno/fimpgo/fimptype"
)

// Specification creates a service specification.
func Specification(
	resourceName,
	resourceAddress,
	address string,
	groups []string,
	events []string,
) *fimptype.Service {
	return &fimptype.Service{
		Address: fmt.Sprintf("/rt:dev/rn:%s/ad:%s/sv:%s/ad:%s", resourceName, resourceAddress, AlarmSystem, address),
		Name:    AlarmSystem,
		Groups:  groups,
		Enabled: true,
		Props: map[string]any{
			PropertySupportedEvents: events,
		},
		Interfaces: requiredInterfaces(),
	}
}

// requiredInterfaces returns required interfaces by the service.
func requiredInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdAlarmGetReport,
			ValueType: fimptype.VTypeStrMap,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtAlarmReport,
			ValueType: fimptype.VTypeStrMap,
			Version:   "1",
		},
	}
}
