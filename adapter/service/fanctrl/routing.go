package fanctrl

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

const (
	CmdModeGetReport = "cmd.mode.get_report"
	CmdModeSet       = "cmd.mode.set"
	EvtModeReport    = "evt.mode.report"

	FanCtrl = "fan_ctrl"
)

func RouteService(serviceRegistry adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		RouteCmdModeSet(serviceRegistry),
		RouteCmdModeGetReport(serviceRegistry),
	}
}

func RouteCmdModeSet(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdModeSet(serviceRegistry),
		router.ForService(FanCtrl),
		router.ForType(CmdModeSet),
	)
}

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

func RouteCmdModeGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdModeGetReport(serviceRegistry),
		router.ForService(FanCtrl),
		router.ForType(CmdModeGetReport),
	)
}

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
