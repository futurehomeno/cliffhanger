package config

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/event"
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
func RouteCmdLogGetLevel(serviceName fimptype.ServiceNameT, logGetter func() string, options ...RoutingOption) *router.Routing {
	return router.NewRouting(
		HandleCmdLogGetLevel(serviceName, logGetter, options...),
		router.ForService(serviceName),
		router.ForType(CmdLogGetLevel),
	)
}

// HandleCmdLogGetLevel returns a handler responsible for handling the command.
func HandleCmdLogGetLevel(serviceName fimptype.ServiceNameT, logGetter func() string, _ ...RoutingOption) router.MessageHandler {
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
func RouteCmdLogSetLevel(serviceName fimptype.ServiceNameT, logSetter func(string) error, options ...RoutingOption) *router.Routing {
	return router.NewRouting(
		HandleCmdLogSetLevel(serviceName, logSetter, options...),
		router.ForService(serviceName),
		router.ForType(CmdLogSetLevel),
	)
}

// HandleCmdLogSetLevel returns a handler responsible for handling the command.
func HandleCmdLogSetLevel(serviceName fimptype.ServiceNameT, logSetter func(string) error, _ ...RoutingOption) router.MessageHandler {
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
			log.Infof("[cliff] Log level updated to %s", logLevel)

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
func RouteCmdConfigGetReport[T any](serviceName fimptype.ServiceNameT, getter func() T, options ...RoutingOption) *router.Routing {
	return router.NewRouting(
		handleCmdConfigGet(serviceName, EvtConfigReport, fimptype.VTypeObject, getter, options...),
		router.ForService(serviceName),
		router.ForType(CmdConfigGetReport),
	)
}

// RouteCmdConfigGetString returns a routing responsible for handling the command.
func RouteCmdConfigGetString[T ~string](serviceName fimptype.ServiceNameT, setting string, getter func() T, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeString, getter, options...)
}

// RouteCmdConfigSetString returns a routing responsible for handling the command.
func RouteCmdConfigSetString[T ~string](serviceName fimptype.ServiceNameT, setting string, setter func(T) error, options ...RoutingOption) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeString, setter, options...)
}

// RouteCmdConfigGetInt returns a routing responsible for handling the command.
func RouteCmdConfigGetInt[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](
	serviceName fimptype.ServiceNameT, setting string, getter func() T, options ...RoutingOption,
) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeInt, getter, options...)
}

// RouteCmdConfigSetInt returns a routing responsible for handling the command.
func RouteCmdConfigSetInt[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](
	serviceName fimptype.ServiceNameT, setting string, setter func(T) error, options ...RoutingOption,
) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeInt, setter, options...)
}

// RouteCmdConfigGetFloat returns a routing responsible for handling the command.
func RouteCmdConfigGetFloat[T ~float64 | ~float32](serviceName fimptype.ServiceNameT, setting string, getter func() T, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeFloat, getter, options...)
}

// RouteCmdConfigSetFloat returns a routing responsible for handling the command.
func RouteCmdConfigSetFloat[T ~float64 | ~float32](serviceName fimptype.ServiceNameT, setting string, setter func(T) error, options ...RoutingOption) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeFloat, setter, options...)
}

// RouteCmdConfigGetBool returns a routing responsible for handling the command.
func RouteCmdConfigGetBool[T ~bool](serviceName fimptype.ServiceNameT, setting string, getter func() T, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeBool, getter, options...)
}

// RouteCmdConfigSetBool returns a routing responsible for handling the command.
func RouteCmdConfigSetBool[T ~bool](serviceName fimptype.ServiceNameT, setting string, setter func(T) error, options ...RoutingOption) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeBool, setter, options...)
}

// RouteCmdConfigGetDuration returns a routing responsible for handling the command.
func RouteCmdConfigGetDuration(serviceName fimptype.ServiceNameT, setting string, getter func() time.Duration, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeString, func() string { return getter().String() }, options...)
}

// RouteCmdConfigSetDuration returns a routing responsible for handling the command.
func RouteCmdConfigSetDuration(serviceName fimptype.ServiceNameT, setting string, rawSetter func(time.Duration) error, options ...RoutingOption) *router.Routing {
	setter := func(value string) error {
		duration, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("config: failed to parse duration: %w", err)
		}

		return rawSetter(duration)
	}

	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeString, setter, options...)
}

// RouteCmdConfigGetStringMap returns a routing responsible for handling the command.
func RouteCmdConfigGetStringMap[M ~map[K]V, K ~string, V ~string](serviceName fimptype.ServiceNameT, setting string, getter func() M, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeStrMap, getter, options...)
}

// RouteCmdConfigSetStringMap returns a routing responsible for handling the command.
func RouteCmdConfigSetStringMap[M ~map[K]V, K ~string, V ~string](serviceName fimptype.ServiceNameT, setting string, setter func(M) error, options ...RoutingOption) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeStrMap, setter, options...)
}

// RouteCmdConfigGetIntMap returns a routing responsible for handling the command.
func RouteCmdConfigGetIntMap[M ~map[K]V, K ~string, V ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](
	serviceName fimptype.ServiceNameT, setting string, getter func() M, options ...RoutingOption,
) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeIntMap, getter, options...)
}

// RouteCmdConfigSetIntMap returns a routing responsible for handling the command.
func RouteCmdConfigSetIntMap[M ~map[K]V, K ~string, V ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](
	serviceName fimptype.ServiceNameT, setting string, setter func(M) error, options ...RoutingOption,
) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeIntMap, setter, options...)
}

// RouteCmdConfigGetFloatMap returns a routing responsible for handling the command.
func RouteCmdConfigGetFloatMap[M ~map[K]V, K ~string, V ~float32 | ~float64](
	serviceName fimptype.ServiceNameT, setting string, getter func() M, options ...RoutingOption,
) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeFloatMap, getter, options...)
}

// RouteCmdConfigSetFloatMap returns a routing responsible for handling the command.
func RouteCmdConfigSetFloatMap[M ~map[K]V, K ~string, V ~float32 | ~float64](
	serviceName fimptype.ServiceNameT, setting string, setter func(M) error, options ...RoutingOption,
) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeFloatMap, setter, options...)
}

// RouteCmdConfigGetBoolMap returns a routing responsible for handling the command.
func RouteCmdConfigGetBoolMap[M ~map[K]V, K ~string, V ~bool](serviceName fimptype.ServiceNameT, setting string, getter func() M, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeBoolMap, getter, options...)
}

// RouteCmdConfigSetBoolMap returns a routing responsible for handling the command.
func RouteCmdConfigSetBoolMap[M ~map[K]V, K ~string, V ~bool](serviceName fimptype.ServiceNameT, setting string, setter func(M) error, options ...RoutingOption) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeBoolMap, setter, options...)
}

// RouteCmdConfigGetStringArray returns a routing responsible for handling the command.
func RouteCmdConfigGetStringArray[T ~string](serviceName fimptype.ServiceNameT, setting string, getter func() []T, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeStrArray, getter, options...)
}

// RouteCmdConfigSetStringArray returns a routing responsible for handling the command.
func RouteCmdConfigSetStringArray[T ~string](serviceName fimptype.ServiceNameT, setting string, setter func([]T) error, options ...RoutingOption) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeStrArray, setter, options...)
}

// RouteCmdConfigGetIntArray returns a routing responsible for handling the command.
func RouteCmdConfigGetIntArray[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](
	serviceName fimptype.ServiceNameT, setting string, getter func() []T, options ...RoutingOption,
) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeIntArray, getter, options...)
}

// RouteCmdConfigSetIntArray returns a routing responsible for handling the command.
func RouteCmdConfigSetIntArray[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](
	serviceName fimptype.ServiceNameT, setting string, setter func([]T) error, options ...RoutingOption,
) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeIntArray, setter, options...)
}

// RouteCmdConfigGetFloatArray returns a routing responsible for handling the command.
func RouteCmdConfigGetFloatArray[T ~float32 | ~float64](serviceName fimptype.ServiceNameT, setting string, getter func() []T, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeFloatArray, getter, options...)
}

// RouteCmdConfigSetFloatArray returns a routing responsible for handling the command.
func RouteCmdConfigSetFloatArray[T ~float32 | ~float64](serviceName fimptype.ServiceNameT, setting string, setter func([]T) error, options ...RoutingOption) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeFloatArray, setter, options...)
}

// RouteCmdConfigGetBoolArray returns a routing responsible for handling the command.
func RouteCmdConfigGetBoolArray[T ~bool](serviceName fimptype.ServiceNameT, setting string, getter func() []T, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeBoolArray, getter, options...)
}

// RouteCmdConfigSetBoolArray returns a routing responsible for handling the command.
func RouteCmdConfigSetBoolArray[T ~bool](serviceName fimptype.ServiceNameT, setting string, setter func([]T) error, options ...RoutingOption) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeBoolArray, setter, options...)
}

// RouteCmdConfigGetObject returns a routing responsible for handling the command.
func RouteCmdConfigGetObject[T any](serviceName fimptype.ServiceNameT, setting string, getter func() T, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeObject, getter, options...)
}

// RouteCmdConfigSetObject returns a routing responsible for handling the command.
func RouteCmdConfigSetObject[T any](serviceName fimptype.ServiceNameT, setting string, setter func(T) error, options ...RoutingOption) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeObject, setter, options...)
}

// routeCmdConfigGet returns a routing responsible for handling the command.
func routeCmdConfigGet[T any](serviceName fimptype.ServiceNameT, setting string, valueType fimptype.ValueTypeT, getter func() T, options ...RoutingOption) *router.Routing {
	return router.NewRouting(
		handleCmdConfigGet(serviceName, fmt.Sprintf(evtConfigReport, setting), valueType, getter, options...),
		router.ForService(serviceName),
		router.ForType(cmdConfigGet+setting),
	)
}

// handleCmdConfigGet returns a handler responsible for handling the command.
func handleCmdConfigGet[T any](serviceName fimptype.ServiceNameT, settingInterface string, valueType fimptype.ValueTypeT, getter func() T, _ ...RoutingOption) router.MessageHandler {
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
func routeCmdConfigSet[T any](serviceName fimptype.ServiceNameT, setting string, valueType fimptype.ValueTypeT, setter func(T) error, options ...RoutingOption) *router.Routing {
	return router.NewRouting(
		handleCmdConfigSet(serviceName, fmt.Sprintf(evtConfigReport, setting), setting, valueType, setter, options...),
		router.ForService(serviceName),
		router.ForType(cmdConfigSet+setting),
	)
}

// handleCmdConfigSet returns a handler responsible for handling the command.
func handleCmdConfigSet[T any](serviceName fimptype.ServiceNameT, settingInterface, setting string, valueType fimptype.ValueTypeT, setter func(T) error, options ...RoutingOption) router.MessageHandler {
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

			opt := getRoutingOptions(options...)
			if opt.eventManager != nil {
				opt.eventManager.Publish(NewConfigurationChangeEvent(serviceName, setting))
			}

			log.WithField("srv", serviceName).
				WithField("param", setting).
				WithField("val", value).
				Info("Cfg changed")

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

// RoutingOption is an interface representing a configuration routing option.
type RoutingOption interface {
	// apply applies the routing option to the routing configuration.
	apply(*routingOptions)
}

// routingOptionFn is a function type that implements the RoutingOption interface.
type routingOptionFn func(*routingOptions)

// apply applies the routing option to the routing configuration.
func (f routingOptionFn) apply(r *routingOptions) {
	f(r)
}

// WithConfigurationChangeEvent returns a routing option that sets the event manager for configuration change events.
func WithConfigurationChangeEvent(eventManager event.Manager) RoutingOption {
	return routingOptionFn(func(configuration *routingOptions) {
		configuration.eventManager = eventManager
	})
}

// routingOptions are options for configuration routing.
type routingOptions struct {
	eventManager event.Manager
}

// getRoutingOptions returns the routing options.
func getRoutingOptions(options ...RoutingOption) *routingOptions {
	o := &routingOptions{}
	for _, option := range options {
		option.apply(o)
	}

	return o
}
