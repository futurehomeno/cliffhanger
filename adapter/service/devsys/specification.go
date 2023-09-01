package devsys

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
	s := &fimptype.Service{
		Address:    fmt.Sprintf("/rt:dev/rn:%s/ad:%s/sv:%s/ad:%s", resourceName, resourceAddress, DevSys, address),
		Name:       DevSys,
		Groups:     groups,
		Enabled:    true,
		Interfaces: requiredInterfaces(),
	}

	return s
}

// requiredInterfaces returns required interfaces by the service.
func requiredInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeOut,
			MsgType:   router.EvtErrorReport,
			ValueType: fimpgo.VTypeString,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   router.EvtSuccessReport,
			ValueType: fimpgo.VTypeNull,
			Version:   "1",
		},
	}
}

// rebootInterfaces returns interfaces used if service supports reboot.
func rebootInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdThingReboot,
			ValueType: fimpgo.VTypeBool,
			Version:   "1",
		},
	}
}
