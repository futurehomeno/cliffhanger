package routing

import (
	"time"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/router"
)

const cmdConfigSet = "cmd.config.set_"

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
