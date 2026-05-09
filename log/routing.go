package log

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/config"
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
	CmdLogSetRevertTimeout    = "cmd.log.set_revert_timeout"
	CmdLogGetRevertTimeout    = "cmd.log.get_revert_timeout"
	EvtLogRevertTimeoutReport = "evt.log.revert_timeout_report"
)

func Route(model any, save func() error) []*router.Routing {
	return []*router.Routing{
		RouteCmdLogGetLevel(fimptype.ServiceNameConfig, GetLogLevel),
	}
}

func RouteCmdLogGetLevel(serviceName fimptype.ServiceNameT, logGetter func() string, options ...config.RoutingOption) *router.Routing {
	return router.NewRouting(
		HandleCmdLogGetLevel(serviceName, logGetter, options...),
		router.ForService(serviceName),
		router.ForType(CmdLogGetLevel),
	)
}

func HandleCmdLogGetLevel(serviceName fimptype.ServiceNameT, logGetter func() string, _ ...config.RoutingOption) router.MessageHandler {
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

func RouteCmdLogSetLevel(serviceName fimptype.ServiceNameT, logSetter func(string) error, options ...config.RoutingOption) *router.Routing {
	return router.NewRouting(
		HandleCmdLogSetLevel(serviceName, logSetter, options...),
		router.ForService(serviceName),
		router.ForType(CmdLogSetLevel),
	)
}

func HandleCmdLogSetLevel(serviceName fimptype.ServiceNameT, logSetter func(string) error, _ ...config.RoutingOption) router.MessageHandler {
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

func RouteCmdLogGetFormat(serviceName fimptype.ServiceNameT, getter func() string, options ...config.RoutingOption) *router.Routing {
	return router.NewRouting(
		HandleCmdLogGetFormat(serviceName, getter, options...),
		router.ForService(serviceName),
		router.ForType(CmdLogGetFormat),
	)
}

func HandleCmdLogGetFormat(serviceName fimptype.ServiceNameT, getter func() string, _ ...config.RoutingOption) router.MessageHandler {
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

func RouteCmdLogSetFormat(serviceName fimptype.ServiceNameT, setter func(string) error, options ...config.RoutingOption) *router.Routing {
	return router.NewRouting(
		HandleCmdLogSetFormat(serviceName, setter, options...),
		router.ForService(serviceName),
		router.ForType(CmdLogSetFormat),
	)
}

func HandleCmdLogSetFormat(serviceName fimptype.ServiceNameT, setter func(string) error, _ ...config.RoutingOption) router.MessageHandler {
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

func RouteCmdLogGetFile(serviceName fimptype.ServiceNameT, getter func() string, options ...config.RoutingOption) *router.Routing {
	return router.NewRouting(
		HandleCmdLogGetFile(serviceName, getter, options...),
		router.ForService(serviceName),
		router.ForType(CmdLogGetFile),
	)
}

func HandleCmdLogGetFile(serviceName fimptype.ServiceNameT, getter func() string, _ ...config.RoutingOption) router.MessageHandler {
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

func RouteCmdLogSetFile(serviceName fimptype.ServiceNameT, setter func(string) error, options ...config.RoutingOption) *router.Routing {
	return router.NewRouting(
		HandleCmdLogSetFile(serviceName, setter, options...),
		router.ForService(serviceName),
		router.ForType(CmdLogSetFile),
	)
}

func HandleCmdLogSetFile(serviceName fimptype.ServiceNameT, setter func(string) error, _ ...config.RoutingOption) router.MessageHandler {
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

func RouteCmdLogGetRevertTimeout(serviceName fimptype.ServiceNameT, getter func() time.Duration, options ...config.RoutingOption) *router.Routing {
	return router.NewRouting(
		HandleCmdLogGetRevertTimeout(serviceName, getter, options...),
		router.ForService(serviceName),
		router.ForType(CmdLogGetRevertTimeout),
	)
}

func HandleCmdLogGetRevertTimeout(serviceName fimptype.ServiceNameT, getter func() time.Duration, _ ...config.RoutingOption) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			return fimpgo.NewStringMessage(
				EvtLogRevertTimeoutReport,
				serviceName,
				getter().String(),
				nil,
				nil,
				message.Payload,
			), nil
		}))
}

func RouteCmdLogSetRevertTimeout(serviceName fimptype.ServiceNameT, setter func(time.Duration) error, options ...config.RoutingOption) *router.Routing {
	return router.NewRouting(
		HandleCmdLogSetRevertTimeout(serviceName, setter, options...),
		router.ForService(serviceName),
		router.ForType(CmdLogSetRevertTimeout),
	)
}

func HandleCmdLogSetRevertTimeout(serviceName fimptype.ServiceNameT, setter func(time.Duration) error, _ ...config.RoutingOption) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			raw, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, err
			}

			d, err := time.ParseDuration(raw)
			if err != nil {
				return nil, fmt.Errorf("log: failed to parse revert timeout %q: %w", raw, err)
			}

			if err := setter(d); err != nil {
				return nil, err
			}

			return fimpgo.NewStringMessage(
				EvtLogRevertTimeoutReport,
				serviceName,
				d.String(),
				nil,
				nil,
				message.Payload,
			), nil
		}))
}

func RoutingForLogManager(serviceName fimptype.ServiceNameT, mgr *LogManager, options ...config.RoutingOption) []*router.Routing {
	return []*router.Routing{
		RouteCmdLogGetLevel(serviceName, mgr.Level, options...),
		RouteCmdLogSetLevel(serviceName, mgr.SetLevel, options...),
		RouteCmdLogGetFormat(serviceName, mgr.Format, options...),
		RouteCmdLogSetFormat(serviceName, mgr.SetFormat, options...),
		RouteCmdLogGetFile(serviceName, mgr.File, options...),
		RouteCmdLogSetFile(serviceName, mgr.SetFile, options...),
		RouteCmdLogGetRevertTimeout(serviceName, mgr.RevertTimeout, options...),
		RouteCmdLogSetRevertTimeout(serviceName, mgr.SetRevertTimeout, options...),
	}
}
