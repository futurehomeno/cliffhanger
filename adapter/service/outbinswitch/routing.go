package outbinswitch

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing service, commands and events.
const (
	CmdBinarySet       = "cmd.binary.set"
	CmdBinaryGetReport = "cmd.binary.get_report"
	EvtBinaryReport    = "evt.binary.report"

	OutBinSwitch = "out_bin_switch"
)

// RouteService returns routing for service specific commands.
func RouteService(serviceRegistry adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		RouteCmdBinarySet(serviceRegistry),
		RouteCmdBinaryGetReport(serviceRegistry),
	}
}

// RouteCmdBinarySet returns a routing responsible for handling the command.
func RouteCmdBinarySet(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdBinarySet(serviceRegistry),
		router.ForService(OutBinSwitch),
		router.ForType(CmdBinarySet),
	)
}

// RouteCmdBinaryGetReport returns a routing responsible for handling the command.
func RouteCmdBinaryGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdBinaryGetReport(serviceRegistry),
		router.ForService(OutBinSwitch),
		router.ForType(CmdBinaryGetReport),
	)
}

// HandleCmdBinarySet returns a handler responsible for handling the command.
func HandleCmdBinarySet(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			outBinSwitch, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			value, err := message.Payload.GetBoolValue()
			if err != nil {
				return nil, fmt.Errorf("failed to get value from the message: %w", err)
			}

			err = outBinSwitch.SetBinaryState(value)
			if err != nil {
				return nil, fmt.Errorf("failed to set state: %w", err)
			}

			_, err = outBinSwitch.SendBinaryReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send binary report: %w", err)
			}

			return nil, nil
		}),
	)
}

// HandleCmdBinaryGetReport returns a handler responsible for handling the command.
func HandleCmdBinaryGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			outBinSwitch, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := outBinSwitch.SendBinaryReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send binary report: %w", err)
			}

			return nil, nil
		}),
	)
}
