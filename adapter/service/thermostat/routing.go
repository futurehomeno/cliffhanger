package thermostat

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

	Thermostat = "thermostat"
)

// RouteService returns routing for service specific commands.
func RouteService(adapter adapter.Adapter) []*router.Routing {
	return []*router.Routing{
		RouteCmdModeSet(adapter),
		RouteCmdSetpointSet(adapter),
		RouteCmdModeGetReport(adapter),
		RouteCmdSetpointGetReport(adapter),
		RouteCmdStateGetReport(adapter),
	}
}

// RouteCmdModeSet returns a routing responsible for handling the command.
func RouteCmdModeSet(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdModeSet(adapter),
		router.ForService(Thermostat),
		router.ForType(CmdModeSet),
	)
}

// HandleCmdModeSet returns a handler responsible for handling the command.
func HandleCmdModeSet(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			thermostat, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			mode, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: provided mode has an incorrect format: %w", err)
			}

			err = thermostat.SetMode(mode)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to set thermostat mode: %w", err)
			}

			_, err = thermostat.SendModeReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send thermostat mode report: %w", err)
			}

			_, err = thermostat.SendSetpointReport(mode, true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send thermostat setpoint report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdSetpointSet returns a routing responsible for handling the command.
func RouteCmdSetpointSet(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdSetpointSet(adapter),
		router.ForService(Thermostat),
		router.ForType(CmdSetpointSet),
	)
}

// HandleCmdSetpointSet returns a handler responsible for handling the command.
func HandleCmdSetpointSet(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			thermostat, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			value, err := message.Payload.GetStrMapValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: provided setpoint string map has an incorrect format: %w", err)
			}

			setpoint, err := SetpointFromStringMap(value)
			if err != nil {
				return nil, fmt.Errorf("adapter: provided setpoint string map has an incorrect format: %w", err)
			}

			err = thermostat.SetSetpoint(setpoint.Mode, setpoint.Temperature, setpoint.Unit)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to set thermostat setpoint: %w", err)
			}

			_, err = thermostat.SendSetpointReport(setpoint.Mode, true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send thermostat setpoint report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdModeGetReport returns a routing responsible for handling the command.
func RouteCmdModeGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdModeGetReport(adapter),
		router.ForService(Thermostat),
		router.ForType(CmdModeGetReport),
	)
}

// HandleCmdModeGetReport returns a handler responsible for handling the command.
func HandleCmdModeGetReport(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			thermostat, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := thermostat.SendModeReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send thermostat mode report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdSetpointGetReport returns a routing responsible for handling the command.
func RouteCmdSetpointGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdSetpointGetReport(adapter),
		router.ForService(Thermostat),
		router.ForType(CmdSetpointGetReport),
	)
}

// HandleCmdSetpointGetReport returns a handler responsible for handling the command.
func HandleCmdSetpointGetReport(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			thermostat, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			mode, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: provided mode has an incorrect format: %w", err)
			}

			_, err = thermostat.SendSetpointReport(mode, true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send thermostat setpoint report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdStateGetReport returns a routing responsible for handling the command.
func RouteCmdStateGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdStateGetReport(adapter),
		router.ForService(Thermostat),
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

			thermostat, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := thermostat.SendStateReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send thermostat state report: %w", err)
			}

			return nil, nil
		}),
	)
}
