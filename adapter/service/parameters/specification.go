package parameters

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
	parameterSizes []int,
) *fimptype.Service {
	s := &fimptype.Service{
		Address:    fmt.Sprintf("/rt:dev/rn:%s/ad:%s/sv:%s/ad:%s", resourceName, resourceAddress, Parameters, address),
		Name:       Parameters,
		Groups:     groups,
		Enabled:    true,
		Interfaces: requiredInterfaces(),
	}

	if len(parameterSizes) > 0 {
		s.Props = map[string]interface{}{
			PropertyParameterSizes: parameterSizes,
		}

		s.Interfaces = append(s.Interfaces, optionalInterfaces()...)
	}

	return s
}

// requiredInterfaces returns required interfaces by the service.
func requiredInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdParamSet,
			ValueType: fimpgo.VTypeObject,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdParamGetReport,
			ValueType: fimpgo.VTypeString,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtParamReport,
			ValueType: fimpgo.VTypeObject,
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

// optionalInterfaces returns optional interfaces by the service.
func optionalInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdSupParamsGetReport,
			ValueType: fimpgo.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtSupParamsReport,
			ValueType: fimpgo.VTypeObject,
			Version:   "1",
		},
	}
}
