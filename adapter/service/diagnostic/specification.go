package diagnostic

import (
	"fmt"

	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/router"
)

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
			MsgType:   EvtRebootsCountReport,
			ValueType: fimptype.VTypeInt,
			Version:   "1",
		},
	}
}

func uptimeInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdUptimeGetReport,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtUptimeReport,
			ValueType: fimptype.VTypeInt,
			Version:   "1",
		},
	}
}

func errorsInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdErrorsGetReport,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtErrorsReport,
			ValueType: fimptype.VTypeStrArray,
			Version:   "1",
		},
	}
}
