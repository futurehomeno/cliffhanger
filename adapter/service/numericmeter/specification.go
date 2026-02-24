package numericmeter

import (
	"fmt"

	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// WithExportUnits adds export units to the service specification.
func WithExportUnits(exportUnits ...Unit) adapter.SpecificationOption {
	return adapter.SpecificationOptionFn(func(f *fimptype.Service) {
		f.Props[PropertySupportedExportUnits] = exportUnits
	})
}

// WithExtendedValues adds extended values to the service specification.
func WithExtendedValues(extendedValues ...Value) adapter.SpecificationOption {
	return adapter.SpecificationOptionFn(func(f *fimptype.Service) {
		f.Props[PropertySupportedExtendedValues] = extendedValues
	})
}

// WithIsVirtual adds is virtual flag to the service specification.
func WithIsVirtual() adapter.SpecificationOption {
	return adapter.SpecificationOptionFn(func(f *fimptype.Service) {
		f.Props[PropertyIsVirtual] = true
	})
}

// Specification creates a service specification.
func Specification(
	serviceName fimptype.ServiceNameT,
	resourceName fimptype.ResourceNameT,
	resourceAddress string,
	address string,
	groups []string,
	units []Unit,
	options ...adapter.SpecificationOption,
) *fimptype.Service {
	specification := &fimptype.Service{
		Address: fmt.Sprintf("/rt:dev/rn:%s/ad:%s/sv:%s/ad:%s", resourceName, resourceAddress, serviceName, address),
		Name:    serviceName,
		Groups:  groups,
		Enabled: true,
		Props: map[string]any{
			PropertySupportedUnits: units,
			PropertyIsVirtual:      false,
		},
		Interfaces: requiredInterfaces(),
	}

	for _, option := range options {
		option.Apply(specification)
	}

	return specification
}

// requiredInterfaces returns required interfaces by the service.
func requiredInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdMeterGetReport,
			ValueType: fimptype.VTypeString,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtMeterReport,
			ValueType: fimptype.VTypeFloat,
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

func resetInterfaces() []fimptype.Interface {
	return []fimptype.Interface{
		{
			Type:      fimptype.TypeIn,
			MsgType:   CmdMeterExportGetReport,
			ValueType: fimptype.VTypeString,
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
			ValueType: fimptype.VTypeString,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtMeterExportReport,
			ValueType: fimptype.VTypeFloat,
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
			ValueType: fimptype.VTypeStrArray,
			Version:   "1",
		},
		{
			Type:      fimptype.TypeOut,
			MsgType:   EvtMeterExtReport,
			ValueType: fimptype.VTypeFloatMap,
			Version:   "1",
		},
	}
}
