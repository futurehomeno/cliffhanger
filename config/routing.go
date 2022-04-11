package config

import (
	"fmt"
	"time"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing commands and events.
const (
	CmdLogSetLevel    = "cmd.log.set_level"
	CmdLogGetLevel    = "cmd.log.get_level"
	EvtLogLevelReport = "evt.log.level_report"

	cmdConfigSet    = "cmd.config.set_"
	cmdConfigGet    = "cmd.config.get_"
	evtConfigReport = "cmd.config.%s_report"
)

// RouteCmdLogGetLevel returns a routing responsible for handling the command.
func RouteCmdLogGetLevel(serviceName string, logGetter func() string) *router.Routing {
	return router.NewRouting(
		HandleCmdLogGetLevel(serviceName, logGetter),
		router.ForService(serviceName),
		router.ForType(CmdLogGetLevel),
	)
}

// HandleCmdLogGetLevel returns a handler responsible for handling the command.
func HandleCmdLogGetLevel(serviceName string, logGetter func() string) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			return fimpgo.NewStringMessage(
				EvtLogLevelReport,
				serviceName,
				logGetter(),
				nil,
				nil,
				message.Payload,
			), nil
		}))
}

// RouteCmdLogSetLevel returns a routing responsible for handling the command.
func RouteCmdLogSetLevel(serviceName string, logSetter func(string) error) *router.Routing {
	return router.NewRouting(
		HandleCmdLogSetLevel(logSetter),
		router.ForService(serviceName),
		router.ForType(CmdLogSetLevel),
	)
}

// HandleCmdLogSetLevel returns a handler responsible for handling the command.
func HandleCmdLogSetLevel(logSetter func(string) error) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			level, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, err
			}

			logLevel, err := log.ParseLevel(level)
			if err != nil {
				return nil, err
			}

			err = logSetter(level)
			if err != nil {
				return nil, err
			}

			log.SetLevel(logLevel)
			log.Infof("Log level updated to %s", logLevel)

			return nil, nil
		}))
}

// RouteCmdConfigGetBool returns a routing responsible for handling the command.
func RouteCmdConfigGetBool(serviceName, setting string, getter func() bool) *router.Routing {
	return router.NewRouting(
		HandleCmdConfigGetBool(serviceName, setting, getter),
		router.ForService(serviceName),
		router.ForType(cmdConfigGet+setting),
	)
}

// HandleCmdConfigGetBool returns a handler responsible for handling the command.
func HandleCmdConfigGetBool(serviceName, setting string, getter func() bool) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			return fimpgo.NewBoolMessage(
				fmt.Sprintf(evtConfigReport, setting),
				serviceName,
				getter(),
				nil,
				nil,
				message.Payload,
			), nil
		}))
}

// RouteCmdConfigSetBool returns a routing responsible for handling the command.
func RouteCmdConfigSetBool(serviceName, setting string, setter func(bool) error) *router.Routing {
	return router.NewRouting(
		HandleCmdConfigSetBool(setter),
		router.ForService(serviceName),
		router.ForType(cmdConfigSet+setting),
	)
}

// HandleCmdConfigSetBool returns a handler responsible for handling the command.
func HandleCmdConfigSetBool(setter func(bool) error) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			value, err := message.Payload.GetBoolValue()
			if err != nil {
				return nil, err
			}

			err = setter(value)
			if err != nil {
				return nil, err
			}

			return nil, nil
		}))
}

// RouteCmdConfigGetString returns a routing responsible for handling the command.
func RouteCmdConfigGetString(serviceName, setting string, getter func() string) *router.Routing {
	return router.NewRouting(
		HandleCmdConfigGetString(serviceName, setting, getter),
		router.ForService(serviceName),
		router.ForType(cmdConfigGet+setting),
	)
}

// HandleCmdConfigGetString returns a handler responsible for handling the command.
func HandleCmdConfigGetString(serviceName, setting string, getter func() string) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			return fimpgo.NewStringMessage(
				fmt.Sprintf(evtConfigReport, setting),
				serviceName,
				getter(),
				nil,
				nil,
				message.Payload,
			), nil
		}))
}

// RouteCmdConfigSetString returns a routing responsible for handling the command.
func RouteCmdConfigSetString(serviceName, setting string, setter func(string) error) *router.Routing {
	return router.NewRouting(
		HandleCmdConfigSetString(setter),
		router.ForService(serviceName),
		router.ForType(cmdConfigSet+setting),
	)
}

// HandleCmdConfigSetString returns a handler responsible for handling the command.
func HandleCmdConfigSetString(setter func(string) error) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			value, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, err
			}

			err = setter(value)
			if err != nil {
				return nil, err
			}

			return nil, nil
		}))
}

// RouteCmdConfigGetInt returns a routing responsible for handling the command.
func RouteCmdConfigGetInt(serviceName, setting string, getter func() int64) *router.Routing {
	return router.NewRouting(
		HandleCmdConfigGetInt(serviceName, setting, getter),
		router.ForService(serviceName),
		router.ForType(cmdConfigGet+setting),
	)
}

// HandleCmdConfigGetInt returns a handler responsible for handling the command.
func HandleCmdConfigGetInt(serviceName, setting string, getter func() int64) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			return fimpgo.NewIntMessage(
				fmt.Sprintf(evtConfigReport, setting),
				serviceName,
				getter(),
				nil,
				nil,
				message.Payload,
			), nil
		}))
}

// RouteCmdConfigSetInt returns a routing responsible for handling the command.
func RouteCmdConfigSetInt(serviceName, setting string, setter func(int) error) *router.Routing {
	return router.NewRouting(
		HandleCmdConfigSetInt(setter),
		router.ForService(serviceName),
		router.ForType(cmdConfigSet+setting),
	)
}

// HandleCmdConfigSetInt returns a handler responsible for handling the command.
func HandleCmdConfigSetInt(setter func(int) error) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			value, err := message.Payload.GetIntValue()
			if err != nil {
				return nil, err
			}

			err = setter(int(value))
			if err != nil {
				return nil, err
			}

			return nil, nil
		}))
}

// RouteCmdConfigGetFloat returns a routing responsible for handling the command.
func RouteCmdConfigGetFloat(serviceName, setting string, getter func() float64) *router.Routing {
	return router.NewRouting(
		HandleCmdConfigGetFloat(serviceName, setting, getter),
		router.ForService(serviceName),
		router.ForType(cmdConfigGet+setting),
	)
}

// HandleCmdConfigGetFloat returns a handler responsible for handling the command.
func HandleCmdConfigGetFloat(serviceName, setting string, getter func() float64) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			return fimpgo.NewFloatMessage(
				fmt.Sprintf(evtConfigReport, setting),
				serviceName,
				getter(),
				nil,
				nil,
				message.Payload,
			), nil
		}))
}

// RouteCmdConfigSetFloat returns a routing responsible for handling the command.
func RouteCmdConfigSetFloat(serviceName, setting string, setter func(float64) error) *router.Routing {
	return router.NewRouting(
		HandleCmdConfigSetFloat(setter),
		router.ForService(serviceName),
		router.ForType(cmdConfigSet+setting),
	)
}

// HandleCmdConfigSetFloat returns a handler responsible for handling the command.
func HandleCmdConfigSetFloat(setter func(float64) error) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			value, err := message.Payload.GetFloatValue()
			if err != nil {
				return nil, err
			}

			err = setter(value)
			if err != nil {
				return nil, err
			}

			return nil, nil
		}))
}

// RouteCmdConfigGetDuration returns a routing responsible for handling the command.
func RouteCmdConfigGetDuration(serviceName, setting string, getter func() time.Duration) *router.Routing {
	return router.NewRouting(
		HandleCmdConfigGetDuration(serviceName, setting, getter),
		router.ForService(serviceName),
		router.ForType(cmdConfigGet+setting),
	)
}

// HandleCmdConfigGetDuration returns a handler responsible for handling the command.
func HandleCmdConfigGetDuration(serviceName, setting string, getter func() time.Duration) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			return fimpgo.NewStringMessage(
				fmt.Sprintf(evtConfigReport, setting),
				serviceName,
				getter().String(),
				nil,
				nil,
				message.Payload,
			), nil
		}))
}

// RouteCmdConfigSetDuration returns a routing responsible for handling the command.
func RouteCmdConfigSetDuration(serviceName, setting string, setter func(time.Duration) error) *router.Routing {
	return router.NewRouting(
		HandleCmdConfigSetDuration(setter),
		router.ForService(serviceName),
		router.ForType(cmdConfigSet+setting),
	)
}

// HandleCmdConfigSetDuration returns a handler responsible for handling the command.
func HandleCmdConfigSetDuration(setter func(time.Duration) error) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			value, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, err
			}

			duration, err := time.ParseDuration(value)
			if err != nil {
				return nil, err
			}

			err = setter(duration)
			if err != nil {
				return nil, err
			}

			return nil, nil
		}))
}
