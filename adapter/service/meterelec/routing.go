package meterelec

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing service, commands and events.
const (
	CmdMeterGetReport    = "cmd.meter.get_report"
	EvtMeterReport       = "evt.meter.report"
	CmdMeterExtGetReport = "cmd.meter_ext.get_report"
	EvtMeterExtReport    = "evt.meter_ext.report"

	MeterElec = "meter_elec"
)

// RouteService returns routing for service specific commands.
func RouteService(adapter adapter.Adapter) []*router.Routing {
	return []*router.Routing{
		RouteCmdMeterGetReport(adapter),
		RouteCmdMeterExtGetReport(adapter),
	}
}

// RouteCmdMeterGetReport returns a routing responsible for handling the command.
func RouteCmdMeterGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdMeterGetReport(adapter),
		router.ForService(MeterElec),
		router.ForType(CmdMeterGetReport),
	)
}

// HandleCmdMeterGetReport returns a handler responsible for handling the command.
func HandleCmdMeterGetReport(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			electricityMeter, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			if message.Payload.ValueType != fimpgo.VTypeString && message.Payload.ValueType != fimpgo.VTypeNull {
				return nil, fmt.Errorf(
					"adapter: provided message value has an invalid type, received %s instead of %s or %s",
					message.Payload.ValueType, fimpgo.VTypeString, fimpgo.VTypeNull,
				)
			}

			units, err := unitsToReport(electricityMeter, message)
			if err != nil {
				return nil, err
			}

			for _, unit := range units {
				_, err = electricityMeter.SendMeterReport(unit, true)
				if err != nil {
					return nil, fmt.Errorf("adapter: failed to send meter report: %w", err)
				}
			}

			return nil, nil
		}),
	)
}

// RouteCmdMeterExtGetReport returns a routing responsible for handling the command.
func RouteCmdMeterExtGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdMeterExtGetReport(adapter),
		router.ForService(MeterElec),
		router.ForType(CmdMeterExtGetReport),
	)
}

// HandleCmdMeterExtGetReport returns a handler responsible for handling the command.
func HandleCmdMeterExtGetReport(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			electricityMeter, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err = electricityMeter.SendMeterExtendedReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send meter extended report: %w", err)
			}

			return nil, nil
		}),
	)
}

// unitsToReport is a helper method that determines which units should be reported.
func unitsToReport(electricityMeter Service, message *fimpgo.Message) ([]string, error) {
	if message.Payload.ValueType != fimpgo.VTypeString && message.Payload.ValueType != fimpgo.VTypeNull {
		return nil, fmt.Errorf(
			"adapter: provided message value has an invalid type, received %s instead of %s or %s",
			message.Payload.ValueType, fimpgo.VTypeString, fimpgo.VTypeNull,
		)
	}

	var units []string

	if message.Payload.ValueType == fimpgo.VTypeNull {
		units = electricityMeter.SupportedUnits()
	} else {
		unit, err := message.Payload.GetStringValue()
		if err != nil {
			return nil, fmt.Errorf("adapter: provided unit has an incorrect format: %w", err)
		}

		if unit != "" {
			units = append(units, unit)
		} else {
			units = electricityMeter.SupportedUnits()
		}
	}

	return units, nil
}
