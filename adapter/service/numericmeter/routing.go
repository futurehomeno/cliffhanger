package numericmeter

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

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

func RouteService(serviceRegistry adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		routeCmdMeterGetReport(serviceRegistry),
		routeCmdMeterExportGetReport(serviceRegistry),
		routeCmdMeterExtGetReport(serviceRegistry),
		routeCmdMeterReset(serviceRegistry),
	}
}

func routeCmdMeterGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdMeterGetReport(serviceRegistry),
		router.ForServicePrefix(prefix),
		router.ForType(CmdMeterGetReport),
	)
}

func handleCmdMeterGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			meter, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			if message.Payload.ValueType != fimptype.VTypeString && message.Payload.ValueType != fimptype.VTypeNull {
				return nil, router.NewError(fimptype.ErrorCodeBadRequest, "provided message value has an invalid type", fmt.Errorf("received %s", message.Payload.ValueType))
			}

			units, err := unitsToReport(message, meter.SupportedUnits())
			if err != nil {
				return nil, err
			}

			for _, unit := range units {
				_, err = meter.SendMeterReport(unit, true)
				if err != nil {
					return nil, fmt.Errorf("failed to send meter report: %w", err)
				}
			}

			return nil, nil
		}),
	)
}

func routeCmdMeterExportGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdMeterExportGetReport(serviceRegistry),
		router.ForServicePrefix(prefix),
		router.ForType(CmdMeterExportGetReport),
	)
}

func handleCmdMeterExportGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			meter, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			if message.Payload.ValueType != fimptype.VTypeString && message.Payload.ValueType != fimptype.VTypeNull {
				return nil, router.NewError(fimptype.ErrorCodeBadRequest, "provided message value has an invalid type", fmt.Errorf("received %s", message.Payload.ValueType))
			}

			units, err := unitsToReport(message, meter.SupportedExportUnits())
			if err != nil {
				return nil, err
			}

			for _, unit := range units {
				_, err = meter.SendMeterExportReport(unit, true)
				if err != nil {
					return nil, fmt.Errorf("failed to send meter report: %w", err)
				}
			}

			return nil, nil
		}),
	)
}

func routeCmdMeterExtGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdMeterExtGetReport(serviceRegistry),
		router.ForServicePrefix(prefix),
		router.ForType(CmdMeterExtGetReport),
	)
}

func handleCmdMeterExtGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			meter, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			if message.Payload.ValueType != fimptype.VTypeStrArray && message.Payload.ValueType != fimptype.VTypeNull {
				return nil, router.NewError(fimptype.ErrorCodeBadRequest, "provided message value has an invalid type", fmt.Errorf("received %s", message.Payload.ValueType))
			}

			values, err := valuesToReport(message, meter.SupportedExtendedValues())
			if err != nil {
				return nil, err
			}

			_, err = meter.SendMeterExtendedReport(values, true)
			if err != nil {
				return nil, fmt.Errorf("failed to send meter extended report: %w", err)
			}

			return nil, nil
		}),
	)
}

func routeCmdMeterReset(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdMeterReset(serviceRegistry),
		router.ForServicePrefix(prefix),
		router.ForType(CmdMeterReset),
	)
}

func handleCmdMeterReset(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			meter, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			err = meter.ResetMeter()
			if err != nil {
				return nil, fmt.Errorf("failed to reset meter: %w", err)
			}

			return nil, nil
		}),
	)
}

// unitsToReport is a helper method that determines which units should be reported.
func unitsToReport(message *fimpgo.Message, supportedUnits Units) (Units, error) {
	if message.Payload.ValueType == fimptype.VTypeNull {
		return supportedUnits, nil
	}

	unit, err := message.Payload.GetStringValue()
	if err != nil {
		return nil, router.NewError(fimptype.ErrorCodeBadRequest, "provided unit has an incorrect format", err)
	}

	if unit == "" {
		return supportedUnits, nil
	}

	return NewUnits(unit), nil
}

// valuesToReport is a helper method that determines which values should be reported.
func valuesToReport(message *fimpgo.Message, supportedValues Values) (Values, error) {
	if message.Payload.ValueType == fimptype.VTypeNull {
		return supportedValues, nil
	}

	values, err := message.Payload.GetStrArrayValue()
	if err != nil {
		return nil, router.NewError(fimptype.ErrorCodeBadRequest, "provided value has an incorrect format", err)
	}

	if len(values) == 0 {
		return supportedValues, nil
	}

	return NewValues(values...), nil
}

// getService returns a service responsible for handling the message.
func getService(serviceRegistry adapter.ServiceRegistry, message *fimpgo.Message) (Service, error) {
	s := serviceRegistry.ServiceByTopic(message.Topic)
	if s == nil {
		return nil, router.NewError(fimptype.ErrorCodeNotFound, "service not found under the provided address", nil)
	}

	numericMeter, ok := s.(Service)
	if !ok {
		return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
	}

	return numericMeter, nil
}
