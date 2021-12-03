package routing

import (
	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/storage"
)

const (
	CmdAppGetManifest          = "cmd.app.get_manifest"
	EvtAppManifestReport       = "evt.app.manifest_report"
	CmdAppGetState             = "cmd.app.get_state"
	EvtAppStateReport          = "evt.config.extended_report"
	CmdConfigGetExtendedReport = "cmd.config.get_extended_report"
	EvtConfigExtendedReport    = "evt.config.extended_report"
)

// RouteCmdAppGetState returns a routing responsible for handling the command.
func RouteCmdAppGetState(serviceName string, appLifecycle lifecycle.Lifecycle) *router.Routing {
	return router.NewRouting(
		HandleCmdAppGetState(serviceName, appLifecycle),
		router.ForService(serviceName),
		router.ForType(CmdAppGetState),
	)
}

// HandleCmdAppGetState returns a handler responsible for handling the command.
func HandleCmdAppGetState(serviceName string, appLifecycle lifecycle.Lifecycle) router.MessageHandler {
	return router.NewMessageHandler(
		func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
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
		},
	)
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
		func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
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
		},
	)
}
