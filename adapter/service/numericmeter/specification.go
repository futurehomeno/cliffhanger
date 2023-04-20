package numericmeter

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/router"
)

// Specification creates a service specification.
func Specification(
	serviceName,
	resourceName,
	resourceAddress,
	address string,
	groups,
	units []string,
	exportUnits []string,
	extendedValues []string,
	isVirtual bool,
) *fimptype.Service {
	specification := &fimptype.Service{
		Address: fmt.Sprintf("/rt:dev/rn:%s/ad:%s/sv:%s/ad:%s", resourceName, resourceAddress, serviceName, address),
		Name:    serviceName,
		Groups:  groups,
		Enabled: true,
		Props: map[string]interface{}{
			PropertySupportedUnits: units,
			PropertyIsVirtual:      isVirtual,
		},
		Interfaces: requiredInterfaces(),
	}

	if len(exportUnits) > 0 {
		specification.Props[PropertySupportedExportUnits] = exportUnits
	}

	if len(extendedValues) > 0 {
		specification.Props[PropertySupportedExtendedValues] = extendedValues
	}

	return specification
}

// requiredInterfaces returns required interfaces by the service.
func requiredInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdMeterGetReport,
			ValueType: fimpgo.VTypeString,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtMeterReport,
			ValueType: fimpgo.VTypeFloat,
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

func resetInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdMeterExportGetReport,
			ValueType: fimpgo.VTypeString,
			Version:   "1",
		},
	}
}

// exportInterfaces returns interfaces supported by the export capable service.
func exportInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdMeterExportGetReport,
			ValueType: fimpgo.VTypeString,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtMeterExportReport,
			ValueType: fimpgo.VTypeFloat,
			Version:   "1",
		},
	}
}

// extendedInterfaces returns interfaces supported by the extended service.
func extendedInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdMeterExtGetReport,
			ValueType: fimpgo.VTypeStrArray,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtMeterExtReport,
			ValueType: fimpgo.VTypeFloatMap,
			Version:   "1",
		},
	}
}
