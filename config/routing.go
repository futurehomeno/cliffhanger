package config

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing commands and events.
const (
	CmdLogSetLevel     = "cmd.log.set_level"
	CmdLogGetLevel     = "cmd.log.get_level"
	EvtLogLevelReport  = "evt.log.level_report"
	CmdConfigGetReport = "cmd.config.get_report"
	EvtConfigReport    = "evt.config.report"

	cmdConfigSet    = "cmd.config.set_"
	cmdConfigGet    = "cmd.config.get_"
	evtConfigReport = "evt.config.%s_report"
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
		HandleCmdLogSetLevel(serviceName, logSetter),
		router.ForService(serviceName),
		router.ForType(CmdLogSetLevel),
	)
}

// HandleCmdLogSetLevel returns a handler responsible for handling the command.
func HandleCmdLogSetLevel(serviceName string, logSetter func(string) error) router.MessageHandler {
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

			return fimpgo.NewStringMessage(
				EvtLogLevelReport,
				serviceName,
				logLevel.String(),
				nil,
				nil,
				message.Payload,
			), nil
		}))
}

// RouteCmdConfigGetReport returns a routing responsible for handling the command.
func RouteCmdConfigGetReport[T any](serviceName string, getter func() T) *router.Routing {
	return router.NewRouting(
		handleCmdConfigGet(serviceName, EvtConfigReport, fimpgo.VTypeObject, getter),
		router.ForService(serviceName),
		router.ForType(CmdConfigGetReport),
	)
}

// RouteCmdConfigGetString returns a routing responsible for handling the command.
func RouteCmdConfigGetString[T ~string](serviceName, setting string, getter func() T) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimpgo.VTypeString, getter)
}

// RouteCmdConfigSetString returns a routing responsible for handling the command.
func RouteCmdConfigSetString[T ~string](serviceName, setting string, setter func(T) error) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimpgo.VTypeString, setter)
}

// RouteCmdConfigGetInt returns a routing responsible for handling the command.
func RouteCmdConfigGetInt[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](
	serviceName, setting string, getter func() T,
) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimpgo.VTypeInt, getter)
}

// RouteCmdConfigSetInt returns a routing responsible for handling the command.
func RouteCmdConfigSetInt[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](
	serviceName, setting string, setter func(T) error,
) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimpgo.VTypeInt, setter)
}

// RouteCmdConfigGetFloat returns a routing responsible for handling the command.
func RouteCmdConfigGetFloat[T ~float64 | ~float32](serviceName, setting string, getter func() T) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimpgo.VTypeFloat, getter)
}

// RouteCmdConfigSetFloat returns a routing responsible for handling the command.
func RouteCmdConfigSetFloat[T ~float64 | ~float32](serviceName, setting string, setter func(T) error) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimpgo.VTypeFloat, setter)
}

// RouteCmdConfigGetBool returns a routing responsible for handling the command.
func RouteCmdConfigGetBool[T ~bool](serviceName, setting string, getter func() T) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimpgo.VTypeBool, getter)
}

// RouteCmdConfigSetBool returns a routing responsible for handling the command.
func RouteCmdConfigSetBool[T ~bool](serviceName, setting string, setter func(T) error) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimpgo.VTypeBool, setter)
}

// RouteCmdConfigGetDuration returns a routing responsible for handling the command.
func RouteCmdConfigGetDuration(serviceName, setting string, getter func() time.Duration) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimpgo.VTypeString, func() string { return getter().String() })
}

// RouteCmdConfigSetDuration returns a routing responsible for handling the command.
func RouteCmdConfigSetDuration(serviceName, setting string, rawSetter func(time.Duration) error) *router.Routing {
	setter := func(value string) error {
		duration, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("config: failed to parse duration: %w", err)
		}

		return rawSetter(duration)
	}

	return routeCmdConfigSet(serviceName, setting, fimpgo.VTypeString, setter)
}

// RouteCmdConfigGetStringMap returns a routing responsible for handling the command.
func RouteCmdConfigGetStringMap[K ~string, V ~string](serviceName, setting string, getter func() map[K]V) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimpgo.VTypeStrMap, getter)
}

// RouteCmdConfigSetStringMap returns a routing responsible for handling the command.
func RouteCmdConfigSetStringMap[K ~string, V ~string](serviceName, setting string, setter func(map[K]V) error) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimpgo.VTypeStrMap, setter)
}

// RouteCmdConfigGetIntMap returns a routing responsible for handling the command.
func RouteCmdConfigGetIntMap[M ~map[K]V, K ~string, V ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](
	serviceName, setting string, getter func() M,
) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimpgo.VTypeIntMap, getter)
}

// RouteCmdConfigSetIntMap returns a routing responsible for handling the command.
func RouteCmdConfigSetIntMap[K ~string, V ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](
	serviceName, setting string, setter func(map[K]V) error,
) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimpgo.VTypeIntMap, setter)
}

// RouteCmdConfigGetFloatMap returns a routing responsible for handling the command.
func RouteCmdConfigGetFloatMap[K ~string, V ~float32 | ~float64](serviceName, setting string, getter func() map[K]V) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimpgo.VTypeFloatMap, getter)
}

// RouteCmdConfigSetFloatMap returns a routing responsible for handling the command.
func RouteCmdConfigSetFloatMap[K ~string, V ~float32 | ~float64](serviceName, setting string, setter func(map[K]V) error) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimpgo.VTypeFloatMap, setter)
}

// RouteCmdConfigGetBoolMap returns a routing responsible for handling the command.
func RouteCmdConfigGetBoolMap[K ~string, V ~bool](serviceName, setting string, getter func() map[K]V) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimpgo.VTypeBoolMap, getter)
}

// RouteCmdConfigSetBoolMap returns a routing responsible for handling the command.
func RouteCmdConfigSetBoolMap[K ~string, V ~bool](serviceName, setting string, setter func(map[K]V) error) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimpgo.VTypeBoolMap, setter)
}

// RouteCmdConfigGetStringArray returns a routing responsible for handling the command.
func RouteCmdConfigGetStringArray[T ~string](serviceName, setting string, getter func() []T) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimpgo.VTypeStrArray, getter)
}

// RouteCmdConfigSetStringArray returns a routing responsible for handling the command.
func RouteCmdConfigSetStringArray[T ~string](serviceName, setting string, setter func([]T) error) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimpgo.VTypeStrArray, setter)
}

// RouteCmdConfigGetIntArray returns a routing responsible for handling the command.
func RouteCmdConfigGetIntArray[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](
	serviceName, setting string, getter func() []T,
) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimpgo.VTypeIntArray, getter)
}

// RouteCmdConfigSetIntArray returns a routing responsible for handling the command.
func RouteCmdConfigSetIntArray[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](
	serviceName, setting string, setter func([]T) error,
) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimpgo.VTypeIntArray, setter)
}

// RouteCmdConfigGetFloatArray returns a routing responsible for handling the command.
func RouteCmdConfigGetFloatArray[T ~float32 | ~float64](serviceName, setting string, getter func() []T) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimpgo.VTypeFloatArray, getter)
}

// RouteCmdConfigSetFloatArray returns a routing responsible for handling the command.
func RouteCmdConfigSetFloatArray[T ~float32 | ~float64](serviceName, setting string, setter func([]T) error) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimpgo.VTypeFloatArray, setter)
}

// RouteCmdConfigGetBoolArray returns a routing responsible for handling the command.
func RouteCmdConfigGetBoolArray[T ~bool](serviceName, setting string, getter func() []T) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimpgo.VTypeBoolArray, getter)
}

// RouteCmdConfigSetBoolArray returns a routing responsible for handling the command.
func RouteCmdConfigSetBoolArray[T ~bool](serviceName, setting string, setter func([]T) error) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimpgo.VTypeBoolArray, setter)
}

// RouteCmdConfigGetObject returns a routing responsible for handling the command.
func RouteCmdConfigGetObject[T any](serviceName, setting string, getter func() T) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimpgo.VTypeObject, getter)
}

// RouteCmdConfigSetObject returns a routing responsible for handling the command.
func RouteCmdConfigSetObject[T any](serviceName, setting string, setter func(T) error) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimpgo.VTypeObject, setter)
}

// routeCmdConfigGet returns a routing responsible for handling the command.
func routeCmdConfigGet[T any](serviceName, setting, valueType string, getter func() T) *router.Routing {
	return router.NewRouting(
		handleCmdConfigGet(serviceName, fmt.Sprintf(evtConfigReport, setting), valueType, getter),
		router.ForService(serviceName),
		router.ForType(cmdConfigGet+setting),
	)
}

// handleCmdConfigGet returns a handler responsible for handling the command.
func handleCmdConfigGet[T any](serviceName, settingInterface, valueType string, getter func() T) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			value := getter()

			return fimpgo.NewMessage(
				settingInterface,
				serviceName,
				valueType,
				value,
				nil,
				nil,
				message.Payload,
			), nil
		}))
}

// routeCmdConfigSet returns a routing responsible for handling the command.
func routeCmdConfigSet[T any](serviceName, setting, valueType string, setter func(T) error) *router.Routing {
	return router.NewRouting(
		handleCmdConfigSet(serviceName, fmt.Sprintf(evtConfigReport, setting), valueType, setter),
		router.ForService(serviceName),
		router.ForType(cmdConfigSet+setting),
	)
}

// handleCmdConfigSet returns a handler responsible for handling the command.
func handleCmdConfigSet[T any](serviceName, settingInterface, valueType string, setter func(T) error) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			if valueType != message.Payload.ValueType {
				return nil, fmt.Errorf("config: message value type %s does not match the expected type %s", message.Payload.ValueType, valueType)
			}

			value, err := getMessageValue[T](message)
			if err != nil {
				return nil, err
			}

			err = setter(value)
			if err != nil {
				return nil, err
			}

			return fimpgo.NewMessage(
				settingInterface,
				serviceName,
				valueType,
				value,
				nil,
				nil,
				message.Payload,
			), nil
		}))
}

// getMessageValue is a helper that returns the value of the message.
func getMessageValue[T any](message *fimpgo.Message) (value T, err error) {
	b := message.Payload.GetRawObjectValue()

	if message.Payload.Value != nil {
		b, err = json.Marshal(message.Payload.Value)
		if err != nil {
			return value, fmt.Errorf("config: failed to marshal message value: %w", err)
		}
	}

	err = json.Unmarshal(b, &value)
	if err != nil {
		return value, fmt.Errorf("config: failed to unmarshal message value: %w", err)
	}

	return value, nil
}
