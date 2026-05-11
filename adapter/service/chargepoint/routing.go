package chargepoint

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/types"
)

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
	CmdPhaseModeSet             = "cmd.phase_mode.set"
	CmdPhaseModeGetReport       = "cmd.phase_mode.get_report"
	EvtPhaseModeReport          = "evt.phase_mode.report"

	Chargepoint = "chargepoint"
)

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
		routeCmdPhaseModeSet(serviceRegistry),
		routeCmdPhaseModeGetReport(serviceRegistry),
	}
}

func routeCmdChargeStart(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdChargeStart(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdChargeStart),
	)
}

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
				return nil, fmt.Errorf("failed to start charging: %w", err)
			}

			_, err = chargepoint.SendStateReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send chargepoint state report: %w", err)
			}

			_, err = chargepoint.SendCurrentSessionReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send chargepoint current session report: %w", err)
			}

			return nil, nil
		}),
	)
}

func routeCmdChargeStop(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdChargeStop(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdChargeStop),
	)
}

func handleCmdChargeStop(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			chargepoint, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			err = chargepoint.StopCharging()
			if err != nil {
				return nil, fmt.Errorf("failed to stop charging: %w", err)
			}

			_, err = chargepoint.SendStateReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send chargepoint state report: %w", err)
			}

			_, err = chargepoint.SendCurrentSessionReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send chargepoint current session report: %w", err)
			}

			return nil, nil
		}),
	)
}

func routeCmdCableLockSet(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdCableLockSet(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdCableLockSet),
	)
}

func handleCmdCableLockSet(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			chargepoint, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			value, err := message.Payload.GetBoolValue()
			if err != nil {
				return nil, fmt.Errorf("provided cable lock value has an incorrect format: %w", err)
			}

			err = chargepoint.SetCableLock(value)
			if err != nil {
				return nil, fmt.Errorf("failed to set chargepoint cable lock: %w", err)
			}

			_, err = chargepoint.SendCableLockReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send chargepoint cable lock report: %w", err)
			}

			return nil, nil
		}),
	)
}

func routeCmdStateGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdStateGetReport(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdStateGetReport),
	)
}

func handleCmdStateGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			chargepoint, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			_, err = chargepoint.SendStateReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send chargepoint state report: %w", err)
			}

			return nil, nil
		}),
	)
}

func routeCmdCableLockGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdCableLockGetReport(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdCableLockGetReport),
	)
}

func handleCmdCableLockGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			chargepoint, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			_, err = chargepoint.SendCableLockReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send chargepoint cable lock report: %w", err)
			}

			return nil, nil
		}),
	)
}

func routeCmdCurrentSessionSetCurrent(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdCurrentSessionSetCurrent(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdCurrentSessionSetCurrent),
	)
}

func handleCmdCurrentSessionSetCurrent(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			chargepoint, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			current, err := message.Payload.GetIntValue()
			if err != nil {
				return nil, fmt.Errorf("provided current value has an incorrect format: %w", err)
			}

			err = chargepoint.SetOfferedCurrent(current)
			if err != nil {
				return nil, fmt.Errorf("failed to set chargepoint offered current: %w", err)
			}

			_, err = chargepoint.SendCurrentSessionReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send chargepoint current session report: %w", err)
			}

			return nil, nil
		}),
	)
}

func routeCmdCurrentSessionGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdCurrentSessionGetReport(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdCurrentSessionGetReport),
	)
}

func handleCmdCurrentSessionGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			chargepoint, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			_, err = chargepoint.SendCurrentSessionReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send chargepoint current session report: %w", err)
			}

			return nil, nil
		}),
	)
}

func routeCmdMaxCurrentSet(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdMaxCurrentSet(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdMaxCurrentSet),
	)
}

func handleCmdMaxCurrentSet(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			chargepoint, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			current, err := message.Payload.GetIntValue()
			if err != nil {
				return nil, fmt.Errorf("provided current value has an incorrect format: %w", err)
			}

			err = chargepoint.SetMaxCurrent(current)
			if err != nil {
				return nil, fmt.Errorf("failed to set chargepoint max current: %w", err)
			}

			_, err = chargepoint.SendMaxCurrentReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send chargepoint max current report: %w", err)
			}

			return nil, nil
		}),
	)
}

func routeCmdMaxCurrentGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdMaxCurrentGetReport(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdMaxCurrentGetReport),
	)
}

func handleCmdMaxCurrentGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			chargepoint, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			_, err = chargepoint.SendMaxCurrentReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send chargepoint max current report: %w", err)
			}

			return nil, nil
		}),
	)
}

func routeCmdPhaseModeSet(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdPhaseModeSet(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdPhaseModeSet),
	)
}

func handleCmdPhaseModeSet(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			chargepoint, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			phaseMode, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("provided phase mode has an incorrect format: %w", err)
			}

			err = chargepoint.SetPhaseMode(types.PhaseMode(phaseMode))
			if err != nil {
				return nil, fmt.Errorf("failed to set chargepoint phase mode: %w", err)
			}

			_, err = chargepoint.SendPhaseModeReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send chargepoint phase mode report: %w", err)
			}

			return nil, nil
		}),
	)
}

func routeCmdPhaseModeGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdPhaseModeGetReport(serviceRegistry),
		router.ForService(Chargepoint),
		router.ForType(CmdPhaseModeGetReport),
	)
}

func handleCmdPhaseModeGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			chargepoint, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			_, err = chargepoint.SendPhaseModeReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send chargepoint phase mode report: %w", err)
			}

			return nil, nil
		}),
	)
}

// getService returns a service responsible for handling the message.
func getService(serviceRegistry adapter.ServiceRegistry, message *fimpgo.Message) (Service, error) {
	s := serviceRegistry.ServiceByTopic(message.Topic)
	if s == nil {
		return nil, fmt.Errorf("service not found under the provided address: %s", message.Addr.ServiceAddress)
	}

	chargepoint, ok := s.(Service)
	if !ok {
		return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
	}

	return chargepoint, nil
}
