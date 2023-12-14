package virtualmeter

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

const (
	VirtualMeterElec = "virtual_meter_elec"

	PropertyNameUnit = "unit"
)

func RouteService(sr adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		routeCmdMeterAdd(sr),
		routeCmdMeterRemove(sr),
		routeCmdMeterGetReport(sr),
		routeCmdConfigSetInterval(sr),
		routeCmdConfigGetInterval(sr),
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

func routeCmdConfigSetInterval(sr adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdConfigSetInterval(sr),
		router.ForServicePrefix(VirtualMeterElec),
		router.ForType(CmdConfigSetInterval),
	)
}

func routeCmdConfigGetInterval(sr adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdConfigGetInterval(sr),
		router.ForServicePrefix(VirtualMeterElec),
		router.ForType(CmdConfigGetInterval),
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

			if err := srv.SendReport(); err != nil {
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

			if err := srv.SendReport(); err != nil {
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

			if err := srv.SendReport(); err != nil {
				return nil, fmt.Errorf("failed to send virtual meter report: %w", err)
			}

			return nil, nil
		}))
}

func handleCmdConfigSetInterval(sr adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			srv, err := getService(sr, message)
			if err != nil {
				return nil, fmt.Errorf("routing: failed to find service: %w", err)
			}

			interval, err := message.Payload.GetIntValue()
			if err != nil {
				return nil, fmt.Errorf("payload value incorrect, should be int: %w", err)
			}

			if err := srv.SetReportingInterval(int(interval)); err != nil {
				return nil, fmt.Errorf("routing: failed to set duration: %w", err)
			}

			if err := srv.SendReportingInterval(); err != nil {
				return nil, fmt.Errorf("routing: failed to send repoting interval when setting one: %w", err)
			}

			return nil, nil
		}))
}

func handleCmdConfigGetInterval(sr adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			srv, err := getService(sr, message)
			if err != nil {
				return nil, fmt.Errorf("routing: failed to find service: %w", err)
			}

			if err := srv.SendReportingInterval(); err != nil {
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
