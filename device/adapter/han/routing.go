package han

import (
	"fmt"
	"strings"

	"github.com/futurehomeno/fimpgo"

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

// RouteCmdMeterGetReport returns a routing responsible for handling the command.
func RouteCmdMeterGetReport(provider Provider) *router.Routing {
	return router.NewRouting(
		HandleCmdMeterGetReport(provider),
		router.ForService(MeterElec),
		router.ForType(CmdMeterGetReport),
	)
}

// HandleCmdMeterGetReport returns a handler responsible for handling the command.
func HandleCmdMeterGetReport(provider Provider) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			meter := provider.Get(message.Addr.ServiceAddress)
			if meter == nil {
				return nil, fmt.Errorf("no device has been found under address: %s", message.Addr.ServiceAddress)
			}

			unit, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("provided unit has an incorrect format: %w", err)
			}

			normalizedUnit, ok := supportedUnit(unit, meter.GetSupportedUnits())
			if !ok {
				return nil, fmt.Errorf("unsupported unit: %s", unit)
			}

			value, err := meter.GetReport(normalizedUnit)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve reading from the meter: %w", err)
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
		}))
}

func supportedUnit(unit string, units []string) (string, bool) {
	for _, u := range units {
		if strings.EqualFold(unit, u) {
			return u, true
		}
	}

	return "", false
}
