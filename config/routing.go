package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/event"
	"github.com/futurehomeno/cliffhanger/router"
)

const (
	CmdLogSetLevel            = "cmd.log.set_level"
	CmdLogGetLevel            = "cmd.log.get_level"
	EvtLogLevelReport         = "evt.log.level_report"
	CmdLogSetFormat           = "cmd.log.set_format"
	CmdLogGetFormat           = "cmd.log.get_format"
	EvtLogFormatReport        = "evt.log.format_report"
	CmdLogSetFile             = "cmd.log.set_file"
	CmdLogGetFile             = "cmd.log.get_file"
	EvtLogFileReport          = "evt.log.file_report"
	CmdConfigGetReport        = "cmd.config.get_report"
	EvtConfigReport           = "evt.config.report"

	cmdConfigSet    = "cmd.config.set_"
	cmdConfigGet    = "cmd.config.get_"
	evtConfigReport = "evt.config.%s_report"
)

func RouteCmdLogGetLevel(serviceName fimptype.ServiceNameT, logGetter func() string, options ...RoutingOption) *router.Routing {
	return router.NewRouting(
		HandleCmdLogGetLevel(serviceName, logGetter, options...),
		router.ForService(serviceName),
		router.ForType(CmdLogGetLevel),
	)
}

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

func RouteCmdLogSetLevel(serviceName fimptype.ServiceNameT, logSetter func(string) error, options ...RoutingOption) *router.Routing {
	return router.NewRouting(
		HandleCmdLogSetLevel(serviceName, logSetter, options...),
		router.ForService(serviceName),
		router.ForType(CmdLogSetLevel),
	)
}

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

func RouteCmdLogGetFormat(serviceName fimptype.ServiceNameT, getter func() string, options ...RoutingOption) *router.Routing {
	return router.NewRouting(
		HandleCmdLogGetFormat(serviceName, getter, options...),
		router.ForService(serviceName),
		router.ForType(CmdLogGetFormat),
	)
}

func HandleCmdLogGetFormat(serviceName fimptype.ServiceNameT, getter func() string, _ ...RoutingOption) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			return fimpgo.NewStringMessage(
				EvtLogFormatReport,
				serviceName,
				getter(),
				nil,
				nil,
				message.Payload,
			), nil
		}))
}

func RouteCmdLogSetFormat(serviceName fimptype.ServiceNameT, setter func(string) error, options ...RoutingOption) *router.Routing {
	return router.NewRouting(
		HandleCmdLogSetFormat(serviceName, setter, options...),
		router.ForService(serviceName),
		router.ForType(CmdLogSetFormat),
	)
}

func HandleCmdLogSetFormat(serviceName fimptype.ServiceNameT, setter func(string) error, _ ...RoutingOption) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			format, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, err
			}

			if err := setter(format); err != nil {
				return nil, err
			}

			log.Infof("[cliff] Log format updated to %s", format)

			return fimpgo.NewStringMessage(
				EvtLogFormatReport,
				serviceName,
				format,
				nil,
				nil,
				message.Payload,
			), nil
		}))
}

func RouteCmdLogGetFile(serviceName fimptype.ServiceNameT, getter func() string, options ...RoutingOption) *router.Routing {
	return router.NewRouting(
		HandleCmdLogGetFile(serviceName, getter, options...),
		router.ForService(serviceName),
		router.ForType(CmdLogGetFile),
	)
}

func HandleCmdLogGetFile(serviceName fimptype.ServiceNameT, getter func() string, _ ...RoutingOption) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			return fimpgo.NewStringMessage(
				EvtLogFileReport,
				serviceName,
				getter(),
				nil,
				nil,
				message.Payload,
			), nil
		}))
}

func RouteCmdLogSetFile(serviceName fimptype.ServiceNameT, setter func(string) error, options ...RoutingOption) *router.Routing {
	return router.NewRouting(
		HandleCmdLogSetFile(serviceName, setter, options...),
		router.ForService(serviceName),
		router.ForType(CmdLogSetFile),
	)
}

// HandleCmdLogSetFile returns a handler responsible for handling the command.
func HandleCmdLogSetFile(serviceName fimptype.ServiceNameT, setter func(string) error, _ ...RoutingOption) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			file, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, err
			}

			if file == "" || file == "." || file == ".." || file == "/" || filepath.Base(file) != file {
				return nil, fmt.Errorf("log file must be a plain file name, not a path: %s", file)
			}

			if err := setter(file); err != nil {
				return nil, err
			}

			log.Infof("[cliff] Log file updated to %s", file)

			return fimpgo.NewStringMessage(
				EvtLogFileReport,
				serviceName,
				file,
				nil,
				nil,
				message.Payload,
			), nil
		}))
}

func RoutingForLogManager(serviceName fimptype.ServiceNameT, mgr *LogManager, options ...RoutingOption) []*router.Routing {
	return []*router.Routing{
		RouteCmdLogGetLevel(serviceName, mgr.Level, options...),
		RouteCmdLogSetLevel(serviceName, mgr.SetLevel, options...),
		RouteCmdLogGetFormat(serviceName, mgr.Format, options...),
		RouteCmdLogSetFormat(serviceName, mgr.SetFormat, options...),
		RouteCmdLogGetFile(serviceName, mgr.File, options...),
		RouteCmdLogSetFile(serviceName, mgr.SetFile, options...),
	}
}

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
