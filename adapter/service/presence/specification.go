package presence

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
	groups []string,
) *fimptype.Service {
	return &fimptype.Service{
		Address: fmt.Sprintf("/rt:dev/rn:%s/ad:%s/sv:%s/ad:%s", resourceName, resourceAddress, SensorPresence, address),
		Name:    SensorPresence,

		Groups:     groups,
		Enabled:    true,
		Props:      nil,
		Interfaces: requiredInterfaces(),
	}
}

// requiredInterfaces returns required interfaces by the service.
func requiredInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdPresenceGetReport,
			ValueType: fimpgo.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtPresenceReport,
			ValueType: fimpgo.VTypeBool,
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
