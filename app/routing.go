package app

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/auth"
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
	CmdAppUninstall            = "cmd.app.uninstall"
	EvtAppUninstallReport      = "evt.app.uninstall_report"
	CmdAppReset                = "cmd.app.reset"
	EvtAppConfigActionReport   = "evt.app.config_action_report"
	CmdAuthLogin               = "cmd.auth.login"
	CmdAuthLogout              = "cmd.auth.logout"
	CmdAuthSetTokens           = "cmd.auth.set_tokens" // nolint:gosec
	EvtAuthStatusReport        = "evt.auth.status_report"
)

// RouteApp creates routing for an application.
func RouteApp(
	serviceName string,
	appLifecycle *lifecycle.Lifecycle,
	configStorage storage.Storage,
	configFactory func() interface{},
	locker router.MessageHandlerLocker,
	app App,
) []*router.Routing {
	routing := []*router.Routing{
		RouteCmdAppGetState(serviceName, appLifecycle),
		RouteCmdConfigGetExtendedReport(serviceName, configStorage),
		RouteCmdAppGetManifest(serviceName, appLifecycle, configStorage, app),
		RouteCmdConfigExtendedSet(serviceName, appLifecycle, configFactory, app, locker),
		RouteCmdAppUninstall(serviceName, appLifecycle, app, locker),
	}

	resettable, ok := app.(ResettableApp)
	if ok {
		routing = append(routing, RouteCmdAppReset(serviceName, locker, resettable))
	}

	logginable, ok := app.(LogginableApp)
	if ok {
		routing = append(
			routing,
			RouteCmdAuthLogin(serviceName, appLifecycle, locker, logginable),
			RouteCmdAuthLogout(serviceName, appLifecycle, locker, logginable),
		)
	}

	authorizable, ok := app.(AuthorizableApp)
	if ok {
		routing = append(
			routing,
			RouteCmdAuthSetTokens(serviceName, appLifecycle, locker, authorizable),
			RouteCmdAuthLogout(serviceName, appLifecycle, locker, authorizable),
		)
	}

	return routing
}

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
	app App,
) *router.Routing {
	return router.NewRouting(
		HandleCmdAppGetManifest(serviceName, appLifecycle, configStorage, app),
		router.ForService(serviceName),
		router.ForType(CmdAppGetManifest),
	)
}

// HandleCmdAppGetManifest returns a handler responsible for handling the command.
func HandleCmdAppGetManifest(
	serviceName string,
	appLifecycle *lifecycle.Lifecycle,
	configStorage storage.Storage,
	app App,
) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			mode, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("provided value has an incorrect format: %w", err)
			}

			m, err := app.GetManifest()
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
// Provided locker is optional.
func RouteCmdConfigExtendedSet(
	serviceName string,
	appLifecycle *lifecycle.Lifecycle,
	configFactory func() interface{},
	app App,
	locker router.MessageHandlerLocker,
) *router.Routing {
	return router.NewRouting(
		HandleCmdConfigExtendedSet(serviceName, appLifecycle, configFactory, app, locker),
		router.ForService(serviceName),
		router.ForType(CmdConfigExtendedSet),
	)
}

// HandleCmdConfigExtendedSet returns a handler responsible for handling the command.
// Provided locker is optional.
func HandleCmdConfigExtendedSet(
	serviceName string,
	appLifecycle *lifecycle.Lifecycle,
	configFactory func() interface{},
	app App,
	locker router.MessageHandlerLocker,
) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			cfg := configFactory()

			err := message.Payload.GetObjectValue(cfg)
			if err != nil {
				return makeConfigurationReply(serviceName, EvtAppConfigReport, message, appLifecycle, err), nil
			}

			err = app.Configure(cfg)
			if err != nil {
				return makeConfigurationReply(serviceName, EvtAppConfigReport, message, appLifecycle, err), nil
			}

			return makeConfigurationReply(serviceName, EvtAppConfigReport, message, appLifecycle, nil), nil
		}),
		router.WithExternalLock(locker),
	)
}

// RouteCmdAppUninstall returns a routing responsible for handling the command.
// Provided locker is optional.
func RouteCmdAppUninstall(
	serviceName string,
	appLifecycle *lifecycle.Lifecycle,
	app App,
	locker router.MessageHandlerLocker,
) *router.Routing {
	return router.NewRouting(
		HandleCmdAppUninstall(serviceName, appLifecycle, app, locker),
		router.ForService(serviceName),
		router.ForType(CmdAppUninstall),
	)
}

// HandleCmdAppUninstall returns a handler responsible for handling the command.
// Provided locker is optional.
func HandleCmdAppUninstall(
	serviceName string,
	appLifecycle *lifecycle.Lifecycle,
	app App,
	locker router.MessageHandlerLocker,
) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			err := app.Uninstall()
			if err != nil {
				return makeConfigurationReply(serviceName, EvtAppUninstallReport, message, appLifecycle, err), nil
			}

			return makeConfigurationReply(serviceName, EvtAppUninstallReport, message, appLifecycle, nil), nil
		}),
		router.WithExternalLock(locker),
	)
}

// RouteCmdAppReset returns a routing responsible for handling the command.
// Provided locker is optional.
func RouteCmdAppReset(
	serviceName string,
	locker router.MessageHandlerLocker,
	app ResettableApp,
) *router.Routing {
	action := func(_ string) *manifest.ButtonActionResponse {
		err := app.Reset()
		if err != nil {
			log.WithError(err).
				Error("failed to reset the application")

			return &manifest.ButtonActionResponse{
				Operation:       CmdAppReset,
				OperationStatus: "error",
				Next:            "ok",
				ErrorCode:       "",
				ErrorText:       "failed to reset the application",
			}
		}

		return &manifest.ButtonActionResponse{
			Operation:       CmdAppReset,
			OperationStatus: "ok",
			Next:            "reload",
		}
	}

	return router.NewRouting(
		HandleConfigActionCommand(serviceName, action, locker),
		router.ForService(serviceName),
		router.ForType(CmdAppReset),
	)
}

// RouteConfigActionCommand returns a routing responsible for handling the command.
// Provided locker is optional.
func RouteConfigActionCommand(
	serviceName, commandName string,
	action func(parameter string) *manifest.ButtonActionResponse,
	locker router.MessageHandlerLocker,
) *router.Routing {
	return router.NewRouting(
		HandleConfigActionCommand(serviceName, action, locker),
		router.ForService(serviceName),
		router.ForType(commandName),
	)
}

// HandleConfigActionCommand returns a handler responsible for handling the command.
// Provided locker is optional.
func HandleConfigActionCommand(
	serviceName string,
	action func(parameter string) *manifest.ButtonActionResponse,
	locker router.MessageHandlerLocker,
) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			parameter, _ := message.Payload.GetStringValue()

			response := action(parameter)

			return fimpgo.NewMessage(
				EvtAppConfigActionReport,
				serviceName,
				fimpgo.VTypeObject,
				response,
				nil,
				nil,
				message.Payload,
			), nil
		}),
		router.WithExternalLock(locker),
	)
}

// makeConfigurationReply creates configuration reply for an edge application.
func makeConfigurationReply(
	serviceName string,
	messageType string,
	message *fimpgo.Message,
	appLifecycle *lifecycle.Lifecycle,
	err error,
) *fimpgo.FimpMessage {
	configReport := &ConfigurationReport{
		OpStatus: OperationStatusOK,
	}

	if err != nil {
		log.WithError(err).
			WithField("topic", message.Topic).
			WithField("service", message.Payload.Service).
			WithField("type", message.Payload.Type).
			Error("failed to configure the application")

		configReport.OpStatus = OperationStatusError
		configReport.OpError = fmt.Sprintf("failed to configure the application: %s", err)
	}

	configReport.AppState = *appLifecycle.GetAllStates()

	return fimpgo.NewMessage(
		messageType,
		serviceName,
		fimpgo.VTypeObject,
		configReport,
		nil,
		nil,
		message.Payload,
	)
}

// RouteCmdAuthLogin returns a routing responsible for handling the command.
func RouteCmdAuthLogin(
	serviceName string,
	appLifecycle *lifecycle.Lifecycle,
	locker router.MessageHandlerLocker,
	app LogginableApp,
) *router.Routing {
	return router.NewRouting(
		HandleCmdAuthLogin(serviceName, appLifecycle, locker, app),
		router.ForService(serviceName),
		router.ForType(CmdAuthLogin),
	)
}

// HandleCmdAuthLogin returns a handler responsible for handling the command.
func HandleCmdAuthLogin(
	serviceName string,
	appLifecycle *lifecycle.Lifecycle,
	locker router.MessageHandlerLocker,
	app LogginableApp,
) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			credentials := &LoginCredentials{}

			err := message.Payload.GetObjectValue(credentials)
			if err != nil {
				return nil, fmt.Errorf("provided login credentials have an invalid format: %w", err)
			}

			report := &AuthenticationReport{}

			if err = app.Login(credentials); err != nil {
				log.WithError(err).Errorf("application: failed to login")

				report.ErrorText = "failed to login"
			}

			report.Status = string(appLifecycle.AuthState())

			// Compatibility hack for FHX which implemented login flow not in accordance with the specification.
			if err != nil || appLifecycle.AuthState() != lifecycle.AuthStateAuthenticated {
				report.Errors = "failed to login"
			}

			msg := fimpgo.NewMessage(
				EvtAuthStatusReport,
				serviceName,
				fimpgo.VTypeObject,
				report,
				nil,
				nil,
				message.Payload,
			)

			return msg, nil
		}),
		router.WithExternalLock(locker))
}

// RouteCmdAuthSetTokens returns a routing responsible for handling the command.
func RouteCmdAuthSetTokens(
	serviceName string,
	appLifecycle *lifecycle.Lifecycle,
	locker router.MessageHandlerLocker,
	app AuthorizableApp,
) *router.Routing {
	return router.NewRouting(
		HandleCmdAuthSetTokens(serviceName, appLifecycle, locker, app),
		router.ForService(serviceName),
		router.ForType(CmdAuthSetTokens),
	)
}

// HandleCmdAuthSetTokens returns a handler responsible for handling the command.
func HandleCmdAuthSetTokens(
	serviceName string,
	appLifecycle *lifecycle.Lifecycle,
	locker router.MessageHandlerLocker,
	app AuthorizableApp,
) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			credentials := &auth.OAuth2TokenResponse{}

			err := message.Payload.GetObjectValue(credentials)
			if err != nil {
				return nil, fmt.Errorf("provided authorization credentials have an invalid format: %w", err)
			}

			report := &AuthenticationReport{}

			if err = app.Authorize(credentials); err != nil {
				log.WithError(err).Errorf("application: failed to authorize")

				report.ErrorText = "failed to authorize"
			}

			report.Status = string(appLifecycle.AuthState())

			msg := fimpgo.NewMessage(
				EvtAuthStatusReport,
				serviceName,
				fimpgo.VTypeObject,
				report,
				nil,
				nil,
				message.Payload,
			)

			return msg, nil
		}),
		router.WithExternalLock(locker))
}

// RouteCmdAuthLogout returns a routing responsible for handling the command.
func RouteCmdAuthLogout(
	serviceName string,
	appLifecycle *lifecycle.Lifecycle,
	locker router.MessageHandlerLocker,
	app LogoutableApp,
) *router.Routing {
	return router.NewRouting(
		HandleCmdAuthLogout(serviceName, appLifecycle, locker, app),
		router.ForService(serviceName),
		router.ForType(CmdAuthLogout),
	)
}

// HandleCmdAuthLogout returns a handler responsible for handling the command.
func HandleCmdAuthLogout(
	serviceName string,
	appLifecycle *lifecycle.Lifecycle,
	locker router.MessageHandlerLocker,
	app LogoutableApp,
) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			report := &AuthenticationReport{}

			if err := app.Logout(); err != nil {
				log.WithError(err).Errorf("application: failed to logout")

				report.ErrorText = "failed to logout"
			}

			report.Status = string(appLifecycle.AuthState())

			msg := fimpgo.NewMessage(
				EvtAuthStatusReport,
				serviceName,
				fimpgo.VTypeObject,
				report,
				nil,
				nil,
				message.Payload,
			)

			return msg, nil
		}),
		router.WithExternalLock(locker))
}
