package numericmeter

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing service, commands and events.
const (
	CmdMeterGetReport       = "cmd.meter.get_report"
	EvtMeterReport          = "evt.meter.report"
	CmdMeterReset           = "cmd.meter.reset"
	CmdMeterExportGetReport = "cmd.meter_export.get_report"
	EvtMeterExportReport    = "evt.meter_export.report"
	CmdMeterExtGetReport    = "cmd.meter_ext.get_report"
	EvtMeterExtReport       = "evt.meter_ext.report"

	MeterElec    = "meter_elec"
	MeterGas     = "meter_gas"
	MeterWater   = "meter_water"
	MeterHeating = "meter_heating"
	MeterCooling = "meter_cooling"

	prefix = "meter_"
)

// RouteService returns routing for service specific commands.
func RouteService(adapter adapter.Adapter) []*router.Routing {
	return []*router.Routing{
		routeCmdMeterGetReport(adapter),
		routeCmdMeterExportGetReport(adapter),
		routeCmdMeterExtGetReport(adapter),
		routeCmdMeterReset(adapter),
	}
}

// routeCmdMeterGetReport returns a routing responsible for handling the command.
func routeCmdMeterGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		handleCmdMeterGetReport(adapter),
		router.ForServicePrefix(prefix),
		router.ForType(CmdMeterGetReport),
	)
}

// handleCmdMeterGetReport returns a handler responsible for handling the command.
func handleCmdMeterGetReport(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			meter, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			if message.Payload.ValueType != fimpgo.VTypeString && message.Payload.ValueType != fimpgo.VTypeNull {
				return nil, fmt.Errorf(
					"adapter: provided message value has an invalid type, received %s instead of %s or %s",
					message.Payload.ValueType, fimpgo.VTypeString, fimpgo.VTypeNull,
				)
			}

			units, err := unitsToReport(message, meter.SupportedUnits())
			if err != nil {
				return nil, err
			}

			for _, unit := range units {
				_, err = meter.SendMeterReport(unit, true)
				if err != nil {
					return nil, fmt.Errorf("adapter: failed to send meter report: %w", err)
				}
			}

			return nil, nil
		}),
	)
}

// routeCmdMeterExportGetReport returns a routing responsible for handling the command.
func routeCmdMeterExportGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		handleCmdMeterExportGetReport(adapter),
		router.ForServicePrefix(prefix),
		router.ForType(CmdMeterExportGetReport),
	)
}

// handleCmdMeterGetReport returns a handler responsible for handling the command.
func handleCmdMeterExportGetReport(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			meter, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			if message.Payload.ValueType != fimpgo.VTypeString && message.Payload.ValueType != fimpgo.VTypeNull {
				return nil, fmt.Errorf(
					"adapter: provided message value has an invalid type, received %s instead of %s or %s",
					message.Payload.ValueType, fimpgo.VTypeString, fimpgo.VTypeNull,
				)
			}

			units, err := unitsToReport(message, meter.SupportedExportUnits())
			if err != nil {
				return nil, err
			}

			for _, unit := range units {
				_, err = meter.SendMeterExportReport(unit, true)
				if err != nil {
					return nil, fmt.Errorf("adapter: failed to send meter report: %w", err)
				}
			}

			return nil, nil
		}),
	)
}

// routeCmdMeterExtGetReport returns a routing responsible for handling the command.
func routeCmdMeterExtGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		handleCmdMeterExtGetReport(adapter),
		router.ForServicePrefix(prefix),
		router.ForType(CmdMeterExtGetReport),
	)
}

// handleCmdMeterExtGetReport returns a handler responsible for handling the command.
func handleCmdMeterExtGetReport(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			meter, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			if message.Payload.ValueType != fimpgo.VTypeStrArray && message.Payload.ValueType != fimpgo.VTypeNull {
				return nil, fmt.Errorf(
					"adapter: provided message value has an invalid type, received %s instead of %s or %s",
					message.Payload.ValueType, fimpgo.VTypeStrArray, fimpgo.VTypeNull,
				)
			}

			values, err := valuesToReport(message, meter.SupportedExtendedValues())
			if err != nil {
				return nil, err
			}

			_, err = meter.SendMeterExtendedReport(values, true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send meter extended report: %w", err)
			}

			return nil, nil
		}),
	)
}

// routeCmdMeterReset returns a routing responsible for handling the command.
func routeCmdMeterReset(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		handleCmdMeterReset(adapter),
		router.ForServicePrefix(prefix),
		router.ForType(CmdMeterReset),
	)
}

// handleCmdMeterReset returns a handler responsible for handling the command.
func handleCmdMeterReset(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			meter, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			err = meter.ResetMeter()
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to reset meter: %w", err)
			}

			return nil, nil
		}),
	)
}

// unitsToReport is a helper method that determines which units should be reported.
func unitsToReport(message *fimpgo.Message, supportedUnits []string) ([]string, error) {
	if message.Payload.ValueType == fimpgo.VTypeNull {
		return supportedUnits, nil
	}

	unit, err := message.Payload.GetStringValue()
	if err != nil {
		return nil, fmt.Errorf("adapter: provided unit has an incorrect format: %w", err)
	}

	if unit == "" {
		return supportedUnits, nil
	}

	return []string{unit}, nil
}

// valuesToReport is a helper method that determines which values should be reported.
func valuesToReport(message *fimpgo.Message, supportedValues []string) ([]string, error) {
	if message.Payload.ValueType == fimpgo.VTypeNull {
		return supportedValues, nil
	}

	values, err := message.Payload.GetStrArrayValue()
	if err != nil {
		return nil, fmt.Errorf("adapter: provided value has an incorrect format: %w", err)
	}

	if len(values) == 0 {
		return supportedValues, nil
	}

	return values, nil
}