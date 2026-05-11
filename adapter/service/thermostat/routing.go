package thermostat

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

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

func RouteService(serviceRegistry adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		RouteCmdModeSet(serviceRegistry),
		RouteCmdSetpointSet(serviceRegistry),
		RouteCmdModeGetReport(serviceRegistry),
		RouteCmdSetpointGetReport(serviceRegistry),
		RouteCmdStateGetReport(serviceRegistry),
	}
}

func RouteCmdModeSet(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdModeSet(serviceRegistry),
		router.ForService(Thermostat),
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

			thermostat, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			mode, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("provided mode has an incorrect format: %w", err)
			}

			err = thermostat.SetMode(mode)
			if err != nil {
				return nil, fmt.Errorf("failed to set thermostat mode: %w", err)
			}

			_, err = thermostat.SendModeReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send thermostat mode report: %w", err)
			}

			if thermostat.SupportsSetpoint(mode) {
				_, err = thermostat.SendSetpointReport(mode, true)
				if err != nil {
					return nil, fmt.Errorf("failed to send thermostat setpoint report: %w", err)
				}
			}

			return nil, nil
		}),
	)
}

func RouteCmdSetpointSet(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdSetpointSet(serviceRegistry),
		router.ForService(Thermostat),
		router.ForType(CmdSetpointSet),
	)
}

func HandleCmdSetpointSet(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			thermostat, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			value, err := message.Payload.GetStrMapValue()
			if err != nil {
				return nil, fmt.Errorf("provided setpoint string map has an incorrect format: %w", err)
			}

			setpoint, err := SetpointFromStringMap(value)
			if err != nil {
				return nil, fmt.Errorf("provided setpoint string map has an incorrect format: %w", err)
			}

			err = thermostat.SetSetpoint(setpoint.Type, setpoint.Temperature, setpoint.Unit)
			if err != nil {
				return nil, fmt.Errorf("failed to set thermostat setpoint: %w", err)
			}

			_, err = thermostat.SendSetpointReport(setpoint.Type, true)
			if err != nil {
				return nil, fmt.Errorf("failed to send thermostat setpoint report: %w", err)
			}

			return nil, nil
		}),
	)
}

func RouteCmdModeGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdModeGetReport(serviceRegistry),
		router.ForService(Thermostat),
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

			thermostat, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := thermostat.SendModeReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send thermostat mode report: %w", err)
			}

			return nil, nil
		}),
	)
}

func RouteCmdSetpointGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdSetpointGetReport(serviceRegistry),
		router.ForService(Thermostat),
		router.ForType(CmdSetpointGetReport),
	)
}

func HandleCmdSetpointGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			thermostat, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			mode, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("provided mode has an incorrect format: %w", err)
			}

			_, err = thermostat.SendSetpointReport(mode, true)
			if err != nil {
				return nil, fmt.Errorf("failed to send thermostat setpoint report: %w", err)
			}

			return nil, nil
		}),
	)
}

func RouteCmdStateGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdStateGetReport(serviceRegistry),
		router.ForService(Thermostat),
		router.ForType(CmdStateGetReport),
	)
}

func HandleCmdStateGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			thermostat, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := thermostat.SendStateReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send thermostat state report: %w", err)
			}

			return nil, nil
		}),
	)
}
