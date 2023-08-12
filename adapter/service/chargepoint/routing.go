package chargepoint

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing service, commands and events.
const (
	CmdChargeStart             = "cmd.charge.start"
	CmdChargeStop              = "cmd.charge.stop"
	CmdStateGetReport          = "cmd.state.get_report"
	EvtStateReport             = "evt.state.report"
	CmdCableLockSet            = "cmd.cable_lock.set"
	CmdCableLockGetReport      = "cmd.cable_lock.get_report"
	EvtCableLockReport         = "evt.cable_lock.report"
	CmdCurrentSessionGetReport = "cmd.current_session.get_report"
	EvtCurrentSessionReport    = "evt.current_session.report"

	Chargepoint = "chargepoint"
)

// RouteService returns routing for service specific commands.
func RouteService(serviceRegistry adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		RouteCmdChargeStart(serviceRegistry),
		RouteCmdChargeStop(serviceRegistry),
		RouteCmdCableLockSet(serviceRegistry),
		RouteCmdStateGetReport(serviceRegistry),
		RouteCmdCurrentSessionGetReport(serviceRegistry),
		RouteCmdCableLockGetReport(serviceRegistry),
	}
}

// RouteCmdChargeStart returns a routing responsible for handling the command.
func RouteCmdChargeStart(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdChargeStart(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdChargeStart),
	)
}

// HandleCmdChargeStart returns a handler responsible for handling the command.
func HandleCmdChargeStart(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			chargepoint, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			mode := message.Payload.Properties[PropertyChargingMode]

			err := chargepoint.StartCharging(mode)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to start charging: %w", err)
			}

			_, err = chargepoint.SendStateReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send chargepoint state report: %w", err)
			}

			_, err = chargepoint.SendCurrentSessionReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send chargepoint current session report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdChargeStop returns a routing responsible for handling the command.
func RouteCmdChargeStop(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdChargeStop(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdChargeStop),
	)
}

// HandleCmdChargeStop returns a handler responsible for handling the command.
func HandleCmdChargeStop(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			chargepoint, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			err := chargepoint.StopCharging()
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to stop charging: %w", err)
			}

			_, err = chargepoint.SendStateReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send chargepoint state report: %w", err)
			}

			_, err = chargepoint.SendCurrentSessionReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send chargepoint current session report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdCableLockSet returns a routing responsible for handling the command.
func RouteCmdCableLockSet(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdCableLockSet(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdCableLockSet),
	)
}

// HandleCmdCableLockSet returns a handler responsible for handling the command.
func HandleCmdCableLockSet(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			chargepoint, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			value, err := message.Payload.GetBoolValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: provided cable lock value has an incorrect format: %w", err)
			}

			err = chargepoint.SetCableLock(value)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to set chargepoint cable lock: %w", err)
			}

			_, err = chargepoint.SendCableLockReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send chargepoint cable lock report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdStateGetReport returns a routing responsible for handling the command.
func RouteCmdStateGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdStateGetReport(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdStateGetReport),
	)
}

// HandleCmdStateGetReport returns a handler responsible for handling the command.
func HandleCmdStateGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			chargepoint, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := chargepoint.SendStateReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send chargepoint state report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdCableLockGetReport returns a routing responsible for handling the command.
func RouteCmdCableLockGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdCableLockGetReport(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdCableLockGetReport),
	)
}

// HandleCmdCableLockGetReport returns a handler responsible for handling the command.
func HandleCmdCableLockGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			chargepoint, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := chargepoint.SendCableLockReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send chargepoint cable lock report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdCurrentSessionGetReport returns a routing responsible for handling the command.
func RouteCmdCurrentSessionGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdCurrentSessionGetReport(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdCurrentSessionGetReport),
	)
}

// HandleCmdCurrentSessionGetReport returns a handler responsible for handling the command.
func HandleCmdCurrentSessionGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			chargepoint, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := chargepoint.SendCurrentSessionReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send chargepoint current session report: %w", err)
			}

			return nil, nil
		}),
	)
}
