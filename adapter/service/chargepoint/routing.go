package chargepoint

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing service, commands and events.
const (
	CmdChargeStart              = "cmd.charge.start"
	CmdChargeStop               = "cmd.charge.stop"
	CmdStateGetReport           = "cmd.state.get_report"
	EvtStateReport              = "evt.state.report"
	CmdCableLockSet             = "cmd.cable_lock.set"
	CmdCableLockGetReport       = "cmd.cable_lock.get_report"
	EvtCableLockReport          = "evt.cable_lock.report"
	CmdCurrentSessionGetReport  = "cmd.current_session.get_report"
	EvtCurrentSessionReport     = "evt.current_session.report"
	CmdCurrentSessionSetCurrent = "cmd.current_session.set_current"
	CmdMaxCurrentSet            = "cmd.max_current.set"
	CmdMaxCurrentGetReport      = "cmd.max_current.get_report"
	EvtMaxCurrentReport         = "evt.max_current.report"

	Chargepoint = "chargepoint"
)

// RouteService returns routing for service specific commands.
func RouteService(serviceRegistry adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		routeCmdChargeStart(serviceRegistry),
		routeCmdChargeStop(serviceRegistry),
		routeCmdStateGetReport(serviceRegistry),
		routeCmdCableLockSet(serviceRegistry),
		routeCmdCableLockGetReport(serviceRegistry),
		routeCmdCurrentSessionSetCurrent(serviceRegistry),
		routeCmdCurrentSessionGetReport(serviceRegistry),
		routeCmdMaxCurrentSet(serviceRegistry),
		routeCmdMaxCurrentGetReport(serviceRegistry),
	}
}

// routeCmdChargeStart returns a routing responsible for handling the command.
func routeCmdChargeStart(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdChargeStart(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdChargeStart),
	)
}

// handleCmdChargeStart returns a handler responsible for handling the command.
func handleCmdChargeStart(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			chargepoint, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			chargingSettings := &ChargingSettings{
				Mode: message.Payload.Properties[PropertyChargingMode],
			}

			err = chargepoint.StartCharging(chargingSettings)
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

// routeCmdChargeStop returns a routing responsible for handling the command.
func routeCmdChargeStop(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdChargeStop(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdChargeStop),
	)
}

// handleCmdChargeStop returns a handler responsible for handling the command.
func handleCmdChargeStop(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			chargepoint, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			err = chargepoint.StopCharging()
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

// routeCmdCableLockSet returns a routing responsible for handling the command.
func routeCmdCableLockSet(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdCableLockSet(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdCableLockSet),
	)
}

// handleCmdCableLockSet returns a handler responsible for handling the command.
func handleCmdCableLockSet(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			chargepoint, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
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

// routeCmdStateGetReport returns a routing responsible for handling the command.
func routeCmdStateGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdStateGetReport(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdStateGetReport),
	)
}

// handleCmdStateGetReport returns a handler responsible for handling the command.
func handleCmdStateGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			chargepoint, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			_, err = chargepoint.SendStateReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send chargepoint state report: %w", err)
			}

			return nil, nil
		}),
	)
}

// routeCmdCableLockGetReport returns a routing responsible for handling the command.
func routeCmdCableLockGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdCableLockGetReport(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdCableLockGetReport),
	)
}

// handleCmdCableLockGetReport returns a handler responsible for handling the command.
func handleCmdCableLockGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			chargepoint, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			_, err = chargepoint.SendCableLockReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send chargepoint cable lock report: %w", err)
			}

			return nil, nil
		}),
	)
}

// routeCmdCurrentSessionSetCurrent returns a routing responsible for handling the command.
func routeCmdCurrentSessionSetCurrent(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdCurrentSessionSetCurrent(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdCurrentSessionSetCurrent),
	)
}

// handleCmdCurrentSessionSetCurrent returns a handler responsible for handling the command.
func handleCmdCurrentSessionSetCurrent(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			chargepoint, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			current, err := message.Payload.GetIntValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: provided current value has an incorrect format: %w", err)
			}

			err = chargepoint.SetOfferedCurrent(current)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to set chargepoint offered current: %w", err)
			}

			_, err = chargepoint.SendCurrentSessionReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send chargepoint current session report: %w", err)
			}

			return nil, nil
		}),
	)
}

// routeCmdCurrentSessionGetReport returns a routing responsible for handling the command.
func routeCmdCurrentSessionGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdCurrentSessionGetReport(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdCurrentSessionGetReport),
	)
}

// handleCmdCurrentSessionGetReport returns a handler responsible for handling the command.
func handleCmdCurrentSessionGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			chargepoint, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			_, err = chargepoint.SendCurrentSessionReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send chargepoint current session report: %w", err)
			}

			return nil, nil
		}),
	)
}

// routeCmdMaxCurrentSet returns a routing responsible for handling the command.
func routeCmdMaxCurrentSet(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdMaxCurrentSet(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdMaxCurrentSet),
	)
}

// handleCmdMaxCurrentSet returns a handler responsible for handling the command.
func handleCmdMaxCurrentSet(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			chargepoint, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			current, err := message.Payload.GetIntValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: provided current value has an incorrect format: %w", err)
			}

			err = chargepoint.SetMaxCurrent(current)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to set chargepoint max current: %w", err)
			}

			_, err = chargepoint.SendMaxCurrentReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send chargepoint max current report: %w", err)
			}

			return nil, nil
		}),
	)
}

// routeCmdMaxCurrentGetReport returns a routing responsible for handling the command.
func routeCmdMaxCurrentGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdMaxCurrentGetReport(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdMaxCurrentSet),
	)
}

// handleCmdMaxCurrentGetReport returns a handler responsible for handling the command.
func handleCmdMaxCurrentGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			chargepoint, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			_, err = chargepoint.SendMaxCurrentReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send chargepoint max current report: %w", err)
			}

			return nil, nil
		}),
	)
}

// getService returns a service responsible for handling the message.
func getService(serviceRegistry adapter.ServiceRegistry, message *fimpgo.Message) (Service, error) {
	s := serviceRegistry.ServiceByTopic(message.Topic)
	if s == nil {
		return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
	}

	chargepoint, ok := s.(Service)
	if !ok {
		return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
	}

	return chargepoint, nil
}
