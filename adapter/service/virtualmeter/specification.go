package virtualmeter

import (
	"fmt"

	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
)

func Specification(
	resourceName fimptype.ResourceNameT,
	resourceAddress string,
	address string,
	groups []string,
	supportedUnits []numericmeter.Unit,
	supportedModes []string,
	options ...adapter.SpecificationOption,
) *fimptype.Service {
	s := &fimptype.Service{
		Address: fmt.Sprintf("/rt:dev/rn:%s/ad:%s/sv:%s/ad:%s", resourceName, resourceAddress, VirtualMeterElec, address),
		Name:    VirtualMeterElec,
		Groups:  groups,
		Enabled: true,
		Props: map[string]interface{}{
			PropertySupportedUnits: supportedUnits,
			PropertySupportedModes: supportedModes,
		},
		Interfaces: requiredInterfaces(),
	}

	for _, o := range options {
		o.Apply(s)
	}

	return s
}

func requiredInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdMeterAdd,
			ValueType: fimptype.VTypeFloatMap,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdMeterRemove,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdMeterGetReport,
			ValueType: fimptype.VTypeNull,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtMeterReport,
			ValueType: fimptype.VTypeFloatMap,
			Version:   "1",
		},
	}
}
