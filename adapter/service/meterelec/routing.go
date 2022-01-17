package meterelec

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing commands and events.
const (
	CmdMeterGetReport    = "cmd.meter.get_report"
	EvtMeterReport       = "evt.meter.reporter"
	CmdMeterExtGetReport = "cmd.meter_ext.get_report"
	EvtMeterExtReport    = "evt.meter_ext.reporter"

	MeterElec = "meter_elec"
)

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

			meterElec, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			unit, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: provided unit has an incorrect format: %w", err)
			}

			value, normalizedUnit, err := meterElec.Report(unit)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to retrieve reporte: %w", err)
			}

			msg := fimpgo.NewFloatMessage(
				EvtMeterReport,
				MeterElec,
				value,
				map[string]string{
					"unit": normalizedUnit,
				},
				nil,
				message.Payload,
			)

			return msg, nil
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

			meterElec, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			report, err := meterElec.ExtendedReport()
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to retrieve extended report: %w", err)
			}

			msg := fimpgo.NewFloatMapMessage(
				EvtMeterExtReport,
				MeterElec,
				report,
				nil,
				nil,
				message.Payload,
			)

			return msg, nil
		}),
	)
}
