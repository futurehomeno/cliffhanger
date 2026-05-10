package config

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/futurehomeno/cliffhanger/event"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"
)

const (
	CmdConfigGetReport = "cmd.config.get_report"
	EvtConfigReport    = "evt.config.report"

	cmdConfigSet    = "cmd.config.set_"
	cmdConfigGet    = "cmd.config.get_"
	evtConfigReport = "evt.config.%s_report"
)

func RouteCmdConfigGetReport[T any](serviceName fimptype.ServiceNameT, getter func() T, options ...RoutingOption) *router.Routing {
	return router.NewRouting(
		handleCmdConfigGet(serviceName, EvtConfigReport, fimptype.VTypeObject, getter, options...),
		router.ForService(serviceName),
		router.ForType(CmdConfigGetReport),
	)
}

func RouteCmdConfigGetString[T ~string](serviceName fimptype.ServiceNameT, setting string, getter func() T, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeString, getter, options...)
}

func RouteCmdConfigSetString[T ~string](serviceName fimptype.ServiceNameT, setting string, setter func(T) error, options ...RoutingOption) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeString, setter, options...)
}

func RouteCmdConfigGetInt[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](
	serviceName fimptype.ServiceNameT, setting string, getter func() T, options ...RoutingOption,
) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeInt, getter, options...)
}

func RouteCmdConfigSetInt[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](
	serviceName fimptype.ServiceNameT, setting string, setter func(T) error, options ...RoutingOption,
) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeInt, setter, options...)
}

func RouteCmdConfigGetFloat[T ~float64 | ~float32](serviceName fimptype.ServiceNameT, setting string, getter func() T, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeFloat, getter, options...)
}

func RouteCmdConfigSetFloat[T ~float64 | ~float32](serviceName fimptype.ServiceNameT, setting string, setter func(T) error, options ...RoutingOption) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeFloat, setter, options...)
}

func RouteCmdConfigGetBool[T ~bool](serviceName fimptype.ServiceNameT, setting string, getter func() T, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeBool, getter, options...)
}

func RouteCmdConfigSetBool[T ~bool](serviceName fimptype.ServiceNameT, setting string, setter func(T) error, options ...RoutingOption) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeBool, setter, options...)
}

func RouteCmdConfigGetDuration(serviceName fimptype.ServiceNameT, setting string, getter func() time.Duration, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeString, func() string { return getter().String() }, options...)
}

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

func RouteCmdConfigGetStringMap[M ~map[K]V, K ~string, V ~string](serviceName fimptype.ServiceNameT, setting string, getter func() M, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeStrMap, getter, options...)
}

func RouteCmdConfigSetStringMap[M ~map[K]V, K ~string, V ~string](serviceName fimptype.ServiceNameT, setting string, setter func(M) error, options ...RoutingOption) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeStrMap, setter, options...)
}

func RouteCmdConfigGetIntMap[M ~map[K]V, K ~string, V ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](
	serviceName fimptype.ServiceNameT, setting string, getter func() M, options ...RoutingOption,
) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeIntMap, getter, options...)
}

func RouteCmdConfigSetIntMap[M ~map[K]V, K ~string, V ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](
	serviceName fimptype.ServiceNameT, setting string, setter func(M) error, options ...RoutingOption,
) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeIntMap, setter, options...)
}

func RouteCmdConfigGetFloatMap[M ~map[K]V, K ~string, V ~float32 | ~float64](
	serviceName fimptype.ServiceNameT, setting string, getter func() M, options ...RoutingOption,
) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeFloatMap, getter, options...)
}

func RouteCmdConfigSetFloatMap[M ~map[K]V, K ~string, V ~float32 | ~float64](
	serviceName fimptype.ServiceNameT, setting string, setter func(M) error, options ...RoutingOption,
) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeFloatMap, setter, options...)
}

func RouteCmdConfigGetBoolMap[M ~map[K]V, K ~string, V ~bool](serviceName fimptype.ServiceNameT, setting string, getter func() M, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeBoolMap, getter, options...)
}

func RouteCmdConfigSetBoolMap[M ~map[K]V, K ~string, V ~bool](serviceName fimptype.ServiceNameT, setting string, setter func(M) error, options ...RoutingOption) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeBoolMap, setter, options...)
}

func RouteCmdConfigGetStringArray[T ~string](serviceName fimptype.ServiceNameT, setting string, getter func() []T, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeStrArray, getter, options...)
}

func RouteCmdConfigSetStringArray[T ~string](serviceName fimptype.ServiceNameT, setting string, setter func([]T) error, options ...RoutingOption) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeStrArray, setter, options...)
}

func RouteCmdConfigGetIntArray[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](
	serviceName fimptype.ServiceNameT, setting string, getter func() []T, options ...RoutingOption,
) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeIntArray, getter, options...)
}

func RouteCmdConfigSetIntArray[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](
	serviceName fimptype.ServiceNameT, setting string, setter func([]T) error, options ...RoutingOption,
) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeIntArray, setter, options...)
}

func RouteCmdConfigGetFloatArray[T ~float32 | ~float64](serviceName fimptype.ServiceNameT, setting string, getter func() []T, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeFloatArray, getter, options...)
}

func RouteCmdConfigSetFloatArray[T ~float32 | ~float64](serviceName fimptype.ServiceNameT, setting string, setter func([]T) error, options ...RoutingOption) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeFloatArray, setter, options...)
}

func RouteCmdConfigGetBoolArray[T ~bool](serviceName fimptype.ServiceNameT, setting string, getter func() []T, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeBoolArray, getter, options...)
}

func RouteCmdConfigSetBoolArray[T ~bool](serviceName fimptype.ServiceNameT, setting string, setter func([]T) error, options ...RoutingOption) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeBoolArray, setter, options...)
}

func RouteCmdConfigGetObject[T any](serviceName fimptype.ServiceNameT, setting string, getter func() T, options ...RoutingOption) *router.Routing {
	return routeCmdConfigGet(serviceName, setting, fimptype.VTypeObject, getter, options...)
}

func RouteCmdConfigSetObject[T any](serviceName fimptype.ServiceNameT, setting string, setter func(T) error, options ...RoutingOption) *router.Routing {
	return routeCmdConfigSet(serviceName, setting, fimptype.VTypeObject, setter, options...)
}

func routeCmdConfigGet[T any](serviceName fimptype.ServiceNameT, setting string, valueType fimptype.ValueTypeT, getter func() T, options ...RoutingOption) *router.Routing {
	return router.NewRouting(
		handleCmdConfigGet(serviceName, fmt.Sprintf(evtConfigReport, setting), valueType, getter, options...),
		router.ForService(serviceName),
		router.ForType(cmdConfigGet+setting),
	)
}

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

func routeCmdConfigSet[T any](serviceName fimptype.ServiceNameT, setting string, valueType fimptype.ValueTypeT, setter func(T) error, options ...RoutingOption) *router.Routing {
	return router.NewRouting(
		handleCmdConfigSet(serviceName, fmt.Sprintf(evtConfigReport, setting), setting, valueType, setter, options...),
		router.ForService(serviceName),
		router.ForType(cmdConfigSet+setting),
	)
}

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

type RoutingOption interface {
	apply(*routingOptions)
}

type routingOptionFn func(*routingOptions)

func (f routingOptionFn) apply(r *routingOptions) {
	f(r)
}

func WithConfigurationChangeEvent(eventManager event.Manager) RoutingOption {
	return routingOptionFn(func(configuration *routingOptions) {
		configuration.eventManager = eventManager
	})
}

type routingOptions struct {
	eventManager event.Manager
}

func getRoutingOptions(options ...RoutingOption) *routingOptions {
	o := &routingOptions{}
	for _, option := range options {
		option.apply(o)
	}

	return o
}
