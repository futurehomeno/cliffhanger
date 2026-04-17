package diagnostic

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
) *fimptype.Service {
	s := &fimptype.Service{
		Address:    fmt.Sprintf("/rt:dev/rn:%s/ad:%s/sv:%s/ad:%s", resourceName, resourceAddress, Diagnostic, address),
		Name:       Diagnostic,
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
			ValueType: fimptype.VTypeString,
			Version:   "1",
		},
	}
}

// lqiInterfaces returns interfaces used if service supports Link Quality Indicator reporting.
func lqiInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdLQIGetReport,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtLQIReport,
			ValueType: fimptype.VTypeInt,
			Version:   "1",
		},
	}
}

// rssiInterfaces returns interfaces used if service supports Received Signal Strength Indicator reporting.
func rssiInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdRSSIGetReport,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtRSSIReport,
			ValueType: fimptype.VTypeInt,
			Version:   "1",
		},
	}
}

// rebootReasonInterfaces returns interfaces used if service supports reboot reason reporting.
func rebootReasonInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdRebootReasonGetReport,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtRebootReasonReport,
			ValueType: fimptype.VTypeString,
			Version:   "1",
		},
	}
}

// rebootsCountInterfaces returns interfaces used if service supports reboots count reporting.
func rebootsCountInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdRebootsCountGetReport,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtRebootCountReport,
			ValueType: fimptype.VTypeInt,
			Version:   "1",
		},
	}
}
