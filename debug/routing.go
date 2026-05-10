package debug

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/sirupsen/logrus"

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

func Route(serviceName fimptype.ServiceNameT, _ ...config.RoutingOption) []*router.Routing {
	if logManager == nil {
		panic("debug: Route called before InitializeLogger")
	}

	return []*router.Routing{
		RouteCmdLogGetLevel(serviceName),
		RouteCmdLogSetLevel(serviceName),
		RouteCmdLogGetFormat(serviceName),
		RouteCmdLogSetFormat(serviceName),
		RouteCmdLogGetFile(serviceName),
		RouteCmdLogSetFile(serviceName),
		RouteCmdLogGetRevertTimeout(serviceName),
		RouteCmdLogSetRevertTimeout(serviceName),
	}
}

func RouteCmdLogGetLevel(serviceName fimptype.ServiceNameT) *router.Routing {
	return router.NewRouting(
		router.NewMessageHandler(
			router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				return fimpgo.NewStringMessage(
					EvtLogLevelReport,
					serviceName,
					logManager.Level(),
					nil,
					nil,
					message.Payload,
				), nil
			})),
		router.ForService(serviceName),
		router.ForType(CmdLogGetLevel),
	)
}

func RouteCmdLogSetLevel(serviceName fimptype.ServiceNameT) *router.Routing {
	return router.NewRouting(
		router.NewMessageHandler(
			router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				level, err := message.Payload.GetStringValue()
				if err != nil {
					return nil, err
				}

				logLevel, err := logrus.ParseLevel(level)
				if err != nil {
					return nil, err
				}

				if err := logManager.store.SetLevel(level); err != nil {
					return nil, err
				}

				if err := logManager.SetLevel(); err != nil {
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
			})),
		router.ForService(serviceName),
		router.ForType(CmdLogSetLevel),
	)
}

func RouteCmdLogGetFormat(serviceName fimptype.ServiceNameT) *router.Routing {
	return router.NewRouting(
		router.NewMessageHandler(
			router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				return fimpgo.NewStringMessage(
					EvtLogFormatReport,
					serviceName,
					logManager.Format(),
					nil,
					nil,
					message.Payload,
				), nil
			})),
		router.ForService(serviceName),
		router.ForType(CmdLogGetFormat),
	)
}

func RouteCmdLogSetFormat(serviceName fimptype.ServiceNameT) *router.Routing {
	return router.NewRouting(
		router.NewMessageHandler(
			router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				format, err := message.Payload.GetStringValue()
				if err != nil {
					return nil, err
				}

				if err := logManager.SetFormat(format); err != nil {
					return nil, err
				}

				logrus.Infof("[cliff] Log format updated to %s", format)

				return fimpgo.NewStringMessage(
					EvtLogFormatReport,
					serviceName,
					format,
					nil,
					nil,
					message.Payload,
				), nil
			})),
		router.ForService(serviceName),
		router.ForType(CmdLogSetFormat),
	)
}

func RouteCmdLogGetFile(serviceName fimptype.ServiceNameT) *router.Routing {
	return router.NewRouting(
		router.NewMessageHandler(
			router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				return fimpgo.NewStringMessage(
					EvtLogFileReport,
					serviceName,
					logManager.File(),
					nil,
					nil,
					message.Payload,
				), nil
			})),
		router.ForService(serviceName),
		router.ForType(CmdLogGetFile),
	)
}

func RouteCmdLogSetFile(serviceName fimptype.ServiceNameT) *router.Routing {
	return router.NewRouting(
		router.NewMessageHandler(
			router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				file, err := message.Payload.GetStringValue()
				if err != nil {
					return nil, err
				}

				if file == "" || file == "." || file == ".." || file == "/" || filepath.Base(file) != file {
					return nil, fmt.Errorf("log file must be a plain file name, not a path: %s", file)
				}

				if err := logManager.SetFile(file); err != nil {
					return nil, err
				}

				logrus.Infof("[cliff] Log file updated to %s", file)

				return fimpgo.NewStringMessage(
					EvtLogFileReport,
					serviceName,
					file,
					nil,
					nil,
					message.Payload,
				), nil
			})),
		router.ForService(serviceName),
		router.ForType(CmdLogSetFile),
	)
}

func RouteCmdLogGetRevertTimeout(serviceName fimptype.ServiceNameT) *router.Routing {
	return router.NewRouting(
		router.NewMessageHandler(
			router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				d := logManager.store.RevertTimeout()
				if d <= 0 {
					d = defaultLogRevertTimeout
				}

				return fimpgo.NewStringMessage(
					EvtLogRevertTimeoutReport,
					serviceName,
					d.String(),
					nil,
					nil,
					message.Payload,
				), nil
			})),
		router.ForService(serviceName),
		router.ForType(CmdLogGetRevertTimeout),
	)
}

func RouteCmdLogSetRevertTimeout(serviceName fimptype.ServiceNameT) *router.Routing {
	return router.NewRouting(
		router.NewMessageHandler(
			router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				raw, err := message.Payload.GetStringValue()
				if err != nil {
					return nil, err
				}

				d, err := time.ParseDuration(raw)
				if err != nil {
					return nil, fmt.Errorf("log: failed to parse revert timeout %q: %w", raw, err)
				}

				if err := logManager.SetRevertTimeout(d); err != nil {
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
			})),
		router.ForService(serviceName),
		router.ForType(CmdLogSetRevertTimeout),
	)
}
