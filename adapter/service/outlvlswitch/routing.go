package outlvlswitch

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing service, commands and events.
const (
	CmdLvlSet       = "cmd.lvl.set"
	CmdLvlGetReport = "cmd.lvl.get_report"
	EvtLvlReport    = "evt.lvl.report"
	CmdLvlStart     = "cmd.lvl.start"
	CmdLvlStop      = "cmd.lvl.stop"
	CmdBinarySet    = "cmd.binary.set"

	OutLvlSwitch = "out_lvl_switch"
)

// RouteService returns routing for service specific commands.
func RouteService(adapter adapter.Adapter) []*router.Routing {
	return []*router.Routing{
		RouteCmdLvlSet(adapter),
		RouteCmdBinarySet(adapter),
		RouteCmdLvlGetReport(adapter),
	}
}

// RouteCmdLvlSet returns a routing responsible for handling the command.
func RouteCmdLvlSet(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdLvlSet(adapter),
		router.ForService(OutLvlSwitch),
		router.ForType(CmdLvlSet),
	)
}

// HandleCmdLvlSet returns a handler responsible for handling the command.
func HandleCmdLvlSet(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			outLvlSwitch, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			lvl, err := message.Payload.GetIntValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: error while getting level value from message: %w", err)
			}

			if outLvlSwitch.SupportDuration() {
				log.Info("The device supports duration")
				if duration, err := message.Payload.Properties.GetIntValue(Duration); err != nil {
					log.Errorf("adapter: error while getting duration value from message: %s. Setting level without duration.", err)

					err = outLvlSwitch.SetLevel(lvl)
					if err != nil {
						return nil, fmt.Errorf("adapter: error while setting level: %w", err)
					}
				} else {
					err = outLvlSwitch.SetLevelWithDuration(lvl, duration)
					if err != nil {
						return nil, fmt.Errorf("adapter: error while setting level with duration: %w", err)
					}
				}
			} else {
				log.Info("The device does not support duration")
				err = outLvlSwitch.SetLevel(lvl)
				if err != nil {
					return nil, fmt.Errorf("adapter: error while setting level: %w", err)
				}
			}

			_, err = outLvlSwitch.SendLevelReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: error while sending level report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdBinarySet returns a routing responsible for handling the command.
func RouteCmdBinarySet(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdBinarySet(adapter),
		router.ForService(OutLvlSwitch),
		router.ForType(CmdBinarySet),
	)
}

// HandleCmdBinarySet returns a handler responsible for handling the command.
func HandleCmdBinarySet(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			outLvlSwitch, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			binary, err := message.Payload.GetBoolValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: error while getting binary value from message: %w", err)
			}

			err = outLvlSwitch.SetBinaryState(binary)
			if err != nil {
				return nil, fmt.Errorf("adapter: error while setting binary: %w", err)
			}

			_, err = outLvlSwitch.SendLevelReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: error while sending level report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdLvlGetReport returns a routing responsible for handling the command.
func RouteCmdLvlGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdLvlGetReport(adapter),
		router.ForService(OutLvlSwitch),
		router.ForType(CmdLvlGetReport),
	)
}

// HandleCmdLvlGetReport returns a handler responsible for handling the command.
func HandleCmdLvlGetReport(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			outLvlSwitch, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := outLvlSwitch.SendLevelReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: error while sending level report: %w", err)
			}

			return nil, nil
		}),
	)
}
