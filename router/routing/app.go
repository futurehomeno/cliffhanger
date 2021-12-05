package routing

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/manifest"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/storage"
)

// Constants defining routing commands and events.
const (
	CmdAppGetManifest          = "cmd.app.get_manifest"
	EvtAppManifestReport       = "evt.app.manifest_report"
	CmdAppGetState             = "cmd.app.get_state"
	EvtAppStateReport          = "evt.app.state_report"
	CmdConfigGetExtendedReport = "cmd.config.get_extended_report"
	EvtConfigExtendedReport    = "evt.config.extended_report"
	CmdConfigExtendedSet       = "cmd.config.extended_set"
	EvtAppConfigReport         = "evt.app.config_report"
)

// RouteCmdAppGetState returns a routing responsible for handling the command.
func RouteCmdAppGetState(serviceName string, appLifecycle *lifecycle.Lifecycle) *router.Routing {
	return router.NewRouting(
		HandleCmdAppGetState(serviceName, appLifecycle),
		router.ForService(serviceName),
		router.ForType(CmdAppGetState),
	)
}

// HandleCmdAppGetState returns a handler responsible for handling the command.
func HandleCmdAppGetState(serviceName string, appLifecycle *lifecycle.Lifecycle) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			msg := fimpgo.NewMessage(
				EvtAppStateReport,
				serviceName,
				fimpgo.VTypeObject,
				appLifecycle.GetAllStates(),
				nil,
				nil,
				message.Payload,
			)

			return msg, nil
		}))
}

// RouteCmdConfigGetExtendedReport returns a routing responsible for handling the command.
func RouteCmdConfigGetExtendedReport(serviceName string, storage storage.Storage) *router.Routing {
	return router.NewRouting(
		HandleCmdConfigGetExtendedReport(serviceName, storage),
		router.ForService(serviceName),
		router.ForType(CmdConfigGetExtendedReport),
	)
}

// HandleCmdConfigGetExtendedReport returns a handler responsible for handling the command.
func HandleCmdConfigGetExtendedReport(serviceName string, storage storage.Storage) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			msg := fimpgo.NewMessage(
				EvtConfigExtendedReport,
				serviceName,
				fimpgo.VTypeObject,
				storage.Model(),
				nil,
				nil,
				message.Payload,
			)

			return msg, nil
		}))
}

// RouteCmdAppGetManifest returns a routing responsible for handling the command.
func RouteCmdAppGetManifest(
	serviceName string,
	appLifecycle *lifecycle.Lifecycle,
	configStorage storage.Storage,
	manifestManager manifest.Manager,
) *router.Routing {
	return router.NewRouting(
		HandleCmdAppGetManifest(serviceName, appLifecycle, configStorage, manifestManager),
		router.ForService(serviceName),
		router.ForType(CmdAppGetManifest),
	)
}

// HandleCmdAppGetManifest returns a handler responsible for handling the command.
func HandleCmdAppGetManifest(
	serviceName string,
	appLifecycle *lifecycle.Lifecycle,
	configStorage storage.Storage,
	manifestManager manifest.Manager,
) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			mode, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("provided value has an incorrect format: %w", err)
			}

			m, err := manifestManager.Get()
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve the manifest: %w", err)
			}

			if mode == "manifest_state" {
				m.AppState = *appLifecycle.GetAllStates()
				m.ConfigState = configStorage.Model()
			}

			reply := fimpgo.NewMessage(EvtAppManifestReport, serviceName, fimpgo.VTypeObject, m, nil, nil, message.Payload)

			return reply, nil
		}),
	)
}

// RouteCmdConfigExtendedSet returns a routing responsible for handling the command.
func RouteCmdConfigExtendedSet(
	serviceName string,
	appLifecycle *lifecycle.Lifecycle,
	configFactory func() interface{},
	manifestManager manifest.Manager,
) *router.Routing {
	return router.NewRouting(
		HandleCmdConfigExtendedSet(serviceName, appLifecycle, configFactory, manifestManager),
		router.ForService(serviceName),
		router.ForType(CmdConfigExtendedSet),
	)
}

// HandleCmdConfigExtendedSet returns a handler responsible for handling the command.
func HandleCmdConfigExtendedSet(
	serviceName string,
	appLifecycle *lifecycle.Lifecycle,
	configFactory func() interface{},
	manifestManager manifest.Manager,
) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			cfg := configFactory()

			err := message.Payload.GetObjectValue(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to parse the configuration object: %w", err)
			}

			configReport := &config.Report{}

			err = manifestManager.Configure(cfg)
			if err != nil {
				log.WithError(err).Error("failed to configure the application")
				configReport.OpStatus = config.OperationStatusError
				configReport.AppState = *appLifecycle.GetAllStates()
				configReport.AppState.LastErrorText = fmt.Sprintf("failed to configure the application: %s", err)
			} else {
				configReport.OpStatus = config.OperationStatusOK
				configReport.AppState = *appLifecycle.GetAllStates()
			}

			reply := fimpgo.NewMessage(EvtAppConfigReport, serviceName, fimpgo.VTypeObject, configReport, nil, nil, message.Payload)

			return reply, nil
		}),
	)
}
