package virtualmeter

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

const (
	VirtualMeterElec = "virtual_meter_elec"

	CmdConfigSetInterval    = "cmd.config.set_interval"
	CmdConfigGetInterval    = "cmd.config.get_interval"
	EvtConfigIntervalReport = "evt.config.interval_report"
	CmdMeterAdd             = "cmd.meter.add"
	CmdMeterRemove          = "cmd.meter.remove"
	CmdMeterGetReport       = "cmd.meter.get_report"
	EvtMeterReport          = "evt.meter.report"

	PropertyNameUnit = "unit"
)

// RouteService returns a routing for the virtual meter service.
func RouteService(sr adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		routeCmdMeterAdd(sr),
		routeCmdMeterRemove(sr),
		routeCmdMeterGetReport(sr),
	}
}

func routeCmdMeterAdd(sr adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdMeterAdd(sr),
		router.ForService(VirtualMeterElec),
		router.ForType(CmdMeterAdd),
	)
}

func routeCmdMeterRemove(sr adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdMeterRemove(sr),
		router.ForService(VirtualMeterElec),
		router.ForType(CmdMeterRemove),
	)
}

func routeCmdMeterGetReport(sr adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdMeterGetReport(sr),
		router.ForService(VirtualMeterElec),
		router.ForType(CmdMeterGetReport),
	)
}

func handleCmdMeterAdd(sr adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			srv, err := getService(sr, message)
			if err != nil {
				return nil, fmt.Errorf("routing: failed to find service: %w", err)
			}

			modes, err := message.Payload.GetFloatMapValue()
			if err != nil {
				return nil, fmt.Errorf("value has incorrect type, expected float map: %w", err)
			}

			unit, ok := message.Payload.Properties.GetStringValue(PropertyNameUnit)
			if !ok {
				return nil, fmt.Errorf("unit property is required in the message")
			}

			if err := srv.AddMeter(modes, unit); err != nil {
				return nil, fmt.Errorf("failed to add meter with modes: %v and unit: %s. %w", modes, unit, err)
			}

			if _, err := srv.SendModesReport(true); err != nil {
				return nil, fmt.Errorf("failed to send virtual meter report: %w", err)
			}

			return nil, nil
		}))
}

func handleCmdMeterRemove(sr adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			srv, err := getService(sr, message)
			if err != nil {
				return nil, fmt.Errorf("routing: failed to find service: %w", err)
			}

			if err := srv.RemoveMeter(); err != nil {
				return nil, fmt.Errorf("failed to remove meter: %w", err)
			}

			if _, err := srv.SendModesReport(true); err != nil {
				return nil, fmt.Errorf("failed to send virtual meter report: %w", err)
			}

			return nil, nil
		}))
}

func handleCmdMeterGetReport(sr adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			srv, err := getService(sr, message)
			if err != nil {
				return nil, fmt.Errorf("routing: failed to find service: %w", err)
			}

			if _, err := srv.SendModesReport(true); err != nil {
				return nil, fmt.Errorf("failed to send virtual meter report: %w", err)
			}

			return nil, nil
		}))
}

// getService returns a service responsible for handling the message.
func getService(serviceRegistry adapter.ServiceRegistry, message *fimpgo.Message) (Service, error) {
	s := serviceRegistry.ServiceByTopic(message.Topic)
	if s == nil {
		return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
	}

	virtialMeter, ok := s.(Service)
	if !ok {
		return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
	}

	return virtialMeter, nil
}
