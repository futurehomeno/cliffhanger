package parameters

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
	return &fimptype.Service{
		Address:    fmt.Sprintf("/rt:dev/rn:%s/ad:%s/sv:%s/ad:%s", resourceName, resourceAddress, Parameters, address),
		Name:       Parameters,
		Groups:     groups,
		Enabled:    true,
		Interfaces: requiredInterfaces(),
	}
}

// requiredInterfaces returns required interfaces by the service.
func requiredInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdSupParamsGetReport,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtSupParamsReport,
			ValueType: fimptype.VTypeObject,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdParamSet,
			ValueType: fimptype.VTypeObject,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdParamGetReport,
			ValueType: fimptype.VTypeString,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtParamReport,
			ValueType: fimptype.VTypeObject,
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
