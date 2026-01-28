package fanctrl

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing service, commands and events.
const (
	CmdModeGetReport = "cmd.mode.get_report"
	CmdModeSet       = "cmd.mode.set"
	EvtModeReport    = "evt.mode.report"

	FanCtrl = "fan_ctrl"
)

// RouteService returns routing for service specific commands.
func RouteService(serviceRegistry adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		RouteCmdModeSet(serviceRegistry),
		RouteCmdModeGetReport(serviceRegistry),
	}
}

// RouteCmdModeSet returns a routing responsible for handling the command.
func RouteCmdModeSet(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdModeSet(serviceRegistry),
		router.ForService(FanCtrl),
		router.ForType(CmdModeSet),
	)
}

// HandleCmdModeSet returns a handler responsible for handling the command.
func HandleCmdModeSet(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			fanctrl, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			mode, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("failed to parse mode: %w", err)
			}

			err = fanctrl.SetMode(mode)
			if err != nil {
				return nil, fmt.Errorf("failed to set mode: %w", err)
			}

			_, err = fanctrl.SendModeReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send mode report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdModeGetReport returns a routing responsible for handling the command.
func RouteCmdModeGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdModeGetReport(serviceRegistry),
		router.ForService(FanCtrl),
		router.ForType(CmdModeGetReport),
	)
}

// HandleCmdModeGetReport returns a handler responsible for handling the command.
func HandleCmdModeGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			modectrl, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := modectrl.SendModeReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send mode report: %w", err)
			}

			return nil, nil
		}),
	)
}
