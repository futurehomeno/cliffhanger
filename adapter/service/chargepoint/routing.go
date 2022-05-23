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
	CmdChargingModeSet         = "cmd.charging_mode.set"
	CmdChargingModeGetReport   = "cmd.charging_mode.get_report"
	EvtChargingModeReport      = "evt.charging_mode.report"

	Chargepoint = "chargepoint"
)

// RouteService returns routing for service specific commands.
func RouteService(adapter adapter.Adapter) []*router.Routing {
	return []*router.Routing{
		RouteCmdChargeStart(adapter),
		RouteCmdChargeStop(adapter),
		RouteCmdCableLockSet(adapter),
		RouteCmdStateGetReport(adapter),
		RouteCmdCurrentSessionGetReport(adapter),
		RouteCmdCableLockGetReport(adapter),
		RouteCmdChargingModeSet(adapter),
		RouteCmdChargingModeGetReport(adapter),
	}
}

// RouteCmdChargeStart returns a routing responsible for handling the command.
func RouteCmdChargeStart(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdChargeStart(adapter),
		router.ForService(Chargepoint),
		router.ForType(CmdChargeStart),
	)
}

// HandleCmdChargeStart returns a handler responsible for handling the command.
func HandleCmdChargeStart(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			chargepoint, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			err := chargepoint.StartCharging()
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
func RouteCmdChargeStop(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdChargeStop(adapter),
		router.ForService(Chargepoint),
		router.ForType(CmdChargeStop),
	)
}

// HandleCmdChargeStop returns a handler responsible for handling the command.
func HandleCmdChargeStop(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
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
func RouteCmdCableLockSet(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdCableLockSet(adapter),
		router.ForService(Chargepoint),
		router.ForType(CmdCableLockSet),
	)
}

// HandleCmdCableLockSet returns a handler responsible for handling the command.
func HandleCmdCableLockSet(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
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
func RouteCmdStateGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdStateGetReport(adapter),
		router.ForService(Chargepoint),
		router.ForType(CmdStateGetReport),
	)
}

// HandleCmdStateGetReport returns a handler responsible for handling the command.
func HandleCmdStateGetReport(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
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
func RouteCmdCableLockGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdCableLockGetReport(adapter),
		router.ForService(Chargepoint),
		router.ForType(CmdCableLockGetReport),
	)
}

// HandleCmdCableLockGetReport returns a handler responsible for handling the command.
func HandleCmdCableLockGetReport(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
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
func RouteCmdCurrentSessionGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdCurrentSessionGetReport(adapter),
		router.ForService(Chargepoint),
		router.ForType(CmdCurrentSessionGetReport),
	)
}

// HandleCmdCurrentSessionGetReport returns a handler responsible for handling the command.
func HandleCmdCurrentSessionGetReport(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
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

// RouteCmdChargingModeSet returns a routing responsible for handling the command.
func RouteCmdChargingModeSet(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdChargingModeSet(adapter),
		router.ForService(Chargepoint),
		router.ForType(CmdChargingModeSet),
	)
}

// HandleCmdChargingModeSet returns a handler responsible for handling the command.
func HandleCmdChargingModeSet(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			chargepoint, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			mode, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: provided charging mode mode has an incorrect format: %w", err)
			}

			if err = chargepoint.SetChargingMode(mode); err != nil {
				return nil, fmt.Errorf("adapter: failed to set charging mode: %w", err)
			}

			_, err = chargepoint.SendChargingModeReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send charging mode report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdChargingModeGetReport returns a routing responsible for handling the command.
func RouteCmdChargingModeGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdChargingModeGetReport(adapter),
		router.ForService(Chargepoint),
		router.ForType(CmdChargingModeGetReport),
	)
}

// HandleCmdChargingModeGetReport returns a handler responsible for handling the command.
func HandleCmdChargingModeGetReport(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			chargepoint, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := chargepoint.SendChargingModeReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send charging mode report: %w", err)
			}

			return nil, nil
		}),
	)
}
