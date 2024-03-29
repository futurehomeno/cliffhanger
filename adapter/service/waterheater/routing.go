package waterheater

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing service, commands and events.
const (
	CmdModeGetReport     = "cmd.mode.get_report"
	CmdModeSet           = "cmd.mode.set"
	EvtModeReport        = "evt.mode.report"
	CmdSetpointGetReport = "cmd.setpoint.get_report"
	CmdSetpointSet       = "cmd.setpoint.set"
	EvtSetpointReport    = "evt.setpoint.report"
	CmdStateGetReport    = "cmd.state.get_report"
	EvtStateReport       = "evt.state.report"

	WaterHeater = "water_heater"
)

// RouteService returns routing for service specific commands.
func RouteService(adapter adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		RouteCmdModeSet(adapter),
		RouteCmdSetpointSet(adapter),
		RouteCmdModeGetReport(adapter),
		RouteCmdSetpointGetReport(adapter),
		RouteCmdStateGetReport(adapter),
	}
}

// RouteCmdModeSet returns a routing responsible for handling the command.
func RouteCmdModeSet(adapter adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdModeSet(adapter),
		router.ForService(WaterHeater),
		router.ForType(CmdModeSet),
	)
}

// HandleCmdModeSet returns a handler responsible for handling the command.
func HandleCmdModeSet(adapter adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			waterHeater, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			mode, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: provided mode has an incorrect format: %w", err)
			}

			err = waterHeater.SetMode(mode)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to set water heater mode: %w", err)
			}

			_, err = waterHeater.SendModeReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send water heater mode report: %w", err)
			}

			if waterHeater.SupportsSetpoint(mode) {
				_, err = waterHeater.SendSetpointReport(mode, true)
				if err != nil {
					return nil, fmt.Errorf("adapter: failed to send water heater setpoint report: %w", err)
				}
			}

			return nil, nil
		}),
	)
}

// RouteCmdSetpointSet returns a routing responsible for handling the command.
func RouteCmdSetpointSet(adapter adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdSetpointSet(adapter),
		router.ForService(WaterHeater),
		router.ForType(CmdSetpointSet),
	)
}

// HandleCmdSetpointSet returns a handler responsible for handling the command.
func HandleCmdSetpointSet(adapter adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			waterHeater, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			setpoint := &Setpoint{}

			err := message.Payload.GetObjectValue(setpoint)
			if err != nil {
				return nil, fmt.Errorf("adapter: provided setpoint object has an incorrect format: %w", err)
			}

			err = waterHeater.SetSetpoint(setpoint.Type, setpoint.Temperature, setpoint.Unit)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to set water heater setpoint: %w", err)
			}

			_, err = waterHeater.SendSetpointReport(setpoint.Type, true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send water heater setpoint report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdModeGetReport returns a routing responsible for handling the command.
func RouteCmdModeGetReport(adapter adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdModeGetReport(adapter),
		router.ForService(WaterHeater),
		router.ForType(CmdModeGetReport),
	)
}

// HandleCmdModeGetReport returns a handler responsible for handling the command.
func HandleCmdModeGetReport(adapter adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			waterHeater, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := waterHeater.SendModeReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send water heater mode report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdSetpointGetReport returns a routing responsible for handling the command.
func RouteCmdSetpointGetReport(adapter adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdSetpointGetReport(adapter),
		router.ForService(WaterHeater),
		router.ForType(CmdSetpointGetReport),
	)
}

// HandleCmdSetpointGetReport returns a handler responsible for handling the command.
func HandleCmdSetpointGetReport(adapter adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			waterHeater, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			mode, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: provided mode has an incorrect format: %w", err)
			}

			_, err = waterHeater.SendSetpointReport(mode, true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send water heater setpoint report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdStateGetReport returns a routing responsible for handling the command.
func RouteCmdStateGetReport(adapter adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdStateGetReport(adapter),
		router.ForService(WaterHeater),
		router.ForType(CmdStateGetReport),
	)
}

// HandleCmdStateGetReport returns a handler responsible for handling the command.
func HandleCmdStateGetReport(adapter adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			waterHeater, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := waterHeater.SendStateReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send water heater state report: %w", err)
			}

			return nil, nil
		}),
	)
}
