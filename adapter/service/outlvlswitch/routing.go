package outlvlswitch

import (
	"fmt"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/pkg/errors"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/utils"
)

const (
	CmdLvlSet       = "cmd.lvl.set"
	CmdLvlGetReport = "cmd.lvl.get_report"
	EvtLvlReport    = "evt.lvl.report"
	CmdLvlStart     = "cmd.lvl.start"
	CmdLvlStop      = "cmd.lvl.stop"
	CmdBinarySet    = "cmd.binary.set"

	OutLvlSwitch = "out_lvl_switch"
)

func RouteService(serviceRegistry adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		RouteCmdLvlSet(serviceRegistry),
		RouteCmdBinarySet(serviceRegistry),
		RouteCmdLvlGetReport(serviceRegistry),
		RouteCmdLvlStart(serviceRegistry),
		RouteCmdLvlStop(serviceRegistry),
	}
}

func RouteCmdLvlSet(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdLvlSet(serviceRegistry),
		router.ForService(OutLvlSwitch),
		router.ForType(CmdLvlSet),
	)
}

func RouteCmdLvlStart(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdLvlStart(serviceRegistry),
		router.ForService(OutLvlSwitch),
		router.ForType(CmdLvlStart),
	)
}

func RouteCmdLvlStop(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdLvlStop(serviceRegistry),
		router.ForService(OutLvlSwitch),
		router.ForType(CmdLvlStop),
	)
}

// HandleCmdLvlStart returns a handler responsible for handling CmdLvlStart message.
func HandleCmdLvlStart(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			service, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			direction, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, errors.Wrap(err, "adapter: error while getting level value from message")
			}

			duration, err := getDurationInSeconds(message)
			if err != nil {
				return nil, errors.Wrap(err, "adapter: error while getting duration value from message")
			}

			startLvl, err := getStartLvl(message)
			if err != nil {
				return nil, errors.Wrap(err, "adapter: error while getting start_lvl value from message")
			}

			if err := service.StartLevelTransition(direction, LevelTransitionParams{StartLvl: startLvl, Duration: duration}); err != nil {
				return nil, errors.Wrap(err, "adapter: failed to start level transitioning")
			}

			_, err = service.SendLevelReport(true)
			if err != nil {
				return nil, errors.Wrap(err, "adapter: error while sending level report")
			}

			return nil, nil
		}),
	)
}

// HandleCmdLvlStop returns a handler responsible for handling CmdLvlStop message.
func HandleCmdLvlStop(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			service, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			if err := service.StopLevelTransition(); err != nil {
				return nil, errors.Wrap(err, "adapter: failed to stop level transitioning")
			}

			_, err := service.SendLevelReport(true)
			if err != nil {
				return nil, errors.Wrap(err, "adapter: error while sending level report")
			}

			return nil, nil
		}),
	)
}

func HandleCmdLvlSet(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			outLvlSwitch, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			lvl, err := message.Payload.GetIntValue()
			if err != nil {
				return nil, fmt.Errorf("error while getting level value from message: %w", err)
			}

			duration, err := getDurationInSeconds(message)
			if err != nil {
				return nil, fmt.Errorf("error while getting duration value from message: %w", err)
			}

			err = outLvlSwitch.SetLevel(lvl, duration)
			if err != nil {
				return nil, fmt.Errorf("error while setting level: %w", err)
			}

			_, err = outLvlSwitch.SendLevelReport(true)
			if err != nil {
				return nil, fmt.Errorf("error while sending level report: %w", err)
			}

			return nil, nil
		}),
	)
}

func RouteCmdBinarySet(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdBinarySet(serviceRegistry),
		router.ForService(OutLvlSwitch),
		router.ForType(CmdBinarySet),
	)
}

func HandleCmdBinarySet(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			outLvlSwitch, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			binary, err := message.Payload.GetBoolValue()
			if err != nil {
				return nil, fmt.Errorf("error while getting binary value from message: %w", err)
			}

			err = outLvlSwitch.SetBinaryState(binary)
			if err != nil {
				return nil, fmt.Errorf("error while setting binary: %w", err)
			}

			_, err = outLvlSwitch.SendLevelReport(true)
			if err != nil {
				return nil, fmt.Errorf("error while sending level report: %w", err)
			}

			return nil, nil
		}),
	)
}

func RouteCmdLvlGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdLvlGetReport(serviceRegistry),
		router.ForService(OutLvlSwitch),
		router.ForType(CmdLvlGetReport),
	)
}

func HandleCmdLvlGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			outLvlSwitch, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := outLvlSwitch.SendLevelReport(true)
			if err != nil {
				return nil, fmt.Errorf("error while sending level report: %w", err)
			}

			return nil, nil
		}),
	)
}

func getStartLvl(message *fimpgo.Message) (*int, error) {
	switch d, ok, err := message.Payload.Properties.GetIntValue(StartLvl); {
	case !ok:
		return nil, nil
	case err != nil:
		return nil, err
	default:
		return utils.Ptr(d), nil
	}
}

func getDurationInSeconds(message *fimpgo.Message) (*time.Duration, error) {
	switch d, ok, err := message.Payload.Properties.GetIntValue(Duration); {
	case !ok:
		return nil, nil
	case err != nil:
		return nil, err
	default:
		return utils.Ptr(time.Duration(d) * time.Second), nil
	}
}
