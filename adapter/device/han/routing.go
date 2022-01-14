package han

import (
	"fmt"
	"strings"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing commands and events.
const (
	CmdMeterGetReport    = "cmd.meter.get_report"
	EvtMeterReport       = "evt.meter.report"
	CmdMeterExtGetReport = "cmd.meter_ext.get_report"
	EvtMeterExtReport    = "evt.meter_ext.report"

	MeterElec = "meter_elec"
)

func Route(adapter adapter.Adapter) []*router.Routing {
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
			thing := adapter.ThingByTopic(message.Topic)
			if thing == nil {
				return nil, fmt.Errorf("adapter: thing not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			meter, ok := thing.(HAN)
			if !ok {
				return nil, fmt.Errorf("adapter: HAN not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			unit, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: provided unit has an incorrect format: %w", err)
			}

			normalizedUnit, ok := supportedUnit(unit, meter.SupportedUnits())
			if !ok {
				return nil, fmt.Errorf("adapter: unit is unsupported by HAN: %s", unit)
			}

			value, err := meter.Report(normalizedUnit)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to retrieve report from HAN: %w", err)
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
			thing := adapter.ThingByTopic(message.Topic)
			if thing == nil {
				return nil, fmt.Errorf("adapter: thing not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			meter, ok := thing.(HAN)
			if !ok {
				return nil, fmt.Errorf("adapter: HAN not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			if !meter.SupportsExtendedReport() {
				return nil, fmt.Errorf("adapter: HAN does not support extended reports")
			}

			report, err := meter.ExtendedReport()
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to retrieve extended report from HAN: %w", err)
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

func supportedUnit(unit string, units []string) (string, bool) {
	for _, u := range units {
		if strings.EqualFold(unit, u) {
			return u, true
		}
	}

	return "", false
}
