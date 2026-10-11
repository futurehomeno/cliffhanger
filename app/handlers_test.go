package app_test

import (
	"errors"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/auth"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/manifest"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/storage"
)

type handlerConfig struct {
	Name   string `json:"name"`
	Secret string `json:"secret"`
}

type redactingStorage struct {
	storage.Storage[*handlerConfig]
}

func (redactingStorage) PublicModel() any { return &handlerConfig{Name: "redacted"} }

type recordingApp struct {
	manifestErr, configureErr, uninstallErr, resetErr, logoutErr, authorizeErr error

	configured   []any
	uninstalls   int
	resets       int
	authorizeArg *auth.OAuth2TokenResponse
}

func (a *recordingApp) GetManifest() (*manifest.Manifest, error) {
	if a.manifestErr != nil {
		return nil, a.manifestErr
	}

	return &manifest.Manifest{InitFlow: []string{"connect"}}, nil
}

func (a *recordingApp) Configure(cfg any) error {
	a.configured = append(a.configured, cfg)

	return a.configureErr
}

func (a *recordingApp) Uninstall() error {
	a.uninstalls++

	return a.uninstallErr
}

func (a *recordingApp) Reset() error {
	a.resets++

	return a.resetErr
}

func (a *recordingApp) Logout() error { return a.logoutErr }

func (a *recordingApp) Authorize(c *auth.OAuth2TokenResponse) error {
	a.authorizeArg = c

	return a.authorizeErr
}

func (a *recordingApp) ErrorsReport() ([]string, error) { return nil, nil }

// newRequest round-trips the command through JSON so object values are available the way they are
// after an MQTT delivery.
func newRequest(t *testing.T, msgType string, value any) *fimpgo.Message {
	t.Helper()

	addr, err := fimpgo.NewAddressFromString("pt:j1/mt:cmd/rt:app/rn:" + testDiagService + "/ad:1")
	require.NoError(t, err)

	var msg *fimpgo.FimpMessage

	switch v := value.(type) {
	case nil:
		msg = fimpgo.NewNullMessage(msgType, testDiagService, nil, nil, nil)
	case string:
		msg = fimpgo.NewStringMessage(msgType, testDiagService, v, nil, nil, nil)
	default:
		msg = fimpgo.NewObjectMessage(msgType, testDiagService, v, nil, nil, nil)
	}

	body, err := msg.SerializeToJson()
	require.NoError(t, err)

	payload, err := fimpgo.NewMessageFromBytes(body)
	require.NoError(t, err)

	return &fimpgo.Message{Topic: addr.Serialize(), Addr: addr, Payload: payload}
}

func requireReply[T any](t *testing.T, reply *fimpgo.Message, wantType string) T {
	t.Helper()

	require.NotNil(t, reply)
	require.Equal(t, wantType, reply.Payload.Interface, "reply props: %v", reply.Payload.Properties)

	v, ok := reply.Payload.Value.(T)
	require.True(t, ok, "unexpected payload %T", reply.Payload.Value)

	return v
}

func requireErrorReport(t *testing.T, reply *fimpgo.Message, wantMsg string) {
	t.Helper()

	require.NotNil(t, reply)
	assert.Equal(t, router.EvtErrorReport, reply.Payload.Interface)
	assert.Contains(t, reply.Payload.Properties[router.PropertyMsg], wantMsg)
}

func TestHandleCmdAppGetManifest_Modes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     any
		wantState bool
	}{
		{name: "null value returns the bare manifest"},
		{name: "unknown mode returns the bare manifest", value: "config"},
		{name: "manifest_state fills the states", value: "manifest_state", wantState: true},
		{name: "full fills the states", value: "full", wantState: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lc := lifecycle.New(nil)
			lc.SetAppHealth(lifecycle.AppHealthRunning, nil)

			st := redactingStorage{storage.New(&handlerConfig{Name: "n", Secret: "s"}, t.TempDir(), "config.json")}
			handler := app.HandleCmdAppGetManifest[*handlerConfig](testDiagService, lc, st, &recordingApp{})

			m := requireReply[*manifest.Manifest](t, handler.Handle(newRequest(t, app.CmdAppGetManifest, tc.value)), app.EvtAppManifestReport)

			assert.Equal(t, []string{"connect"}, m.InitFlow)

			if !tc.wantState {
				assert.Empty(t, m.AppState.Health)
				assert.Nil(t, m.ConfigState)

				return
			}

			assert.Equal(t, string(lifecycle.AppHealthRunning), m.AppState.Health)
			assert.Equal(t, &handlerConfig{Name: "redacted"}, m.ConfigState)
		})
	}
}

func TestHandleCmdAppGetManifest_Errors(t *testing.T) {
	t.Parallel()

	st := storage.New(&handlerConfig{}, t.TempDir(), "config.json")

	t.Run("non-string mode", func(t *testing.T) {
		t.Parallel()

		handler := app.HandleCmdAppGetManifest[*handlerConfig](testDiagService, lifecycle.New(nil), st, &recordingApp{})

		requireErrorReport(t, handler.Handle(newRequest(t, app.CmdAppGetManifest, map[string]string{"a": "b"})), "provided value has an incorrect format")
	})

	t.Run("manifest retrieval fails", func(t *testing.T) {
		t.Parallel()

		handler := app.HandleCmdAppGetManifest[*handlerConfig](testDiagService, lifecycle.New(nil), st, &recordingApp{manifestErr: errors.New("boom")})

		requireErrorReport(t, handler.Handle(newRequest(t, app.CmdAppGetManifest, "full")), "failed to retrieve the manifest")
	})
}

func TestHandleCmdConfigGetExtendedReport_PublicModel(t *testing.T) {
	t.Parallel()

	raw := storage.New(&handlerConfig{Name: "n", Secret: "s"}, t.TempDir(), "config.json")

	reply := app.HandleCmdConfigGetExtendedReport[*handlerConfig](testDiagService, redactingStorage{raw}).
		Handle(newRequest(t, app.CmdConfigGetExtendedReport, nil))
	assert.Equal(t, &handlerConfig{Name: "redacted"}, requireReply[any](t, reply, app.EvtConfigExtendedReport))

	reply = app.HandleCmdConfigGetExtendedReport[*handlerConfig](testDiagService, raw).
		Handle(newRequest(t, app.CmdConfigGetExtendedReport, nil))
	assert.Equal(t, &handlerConfig{Name: "n", Secret: "s"}, requireReply[any](t, reply, app.EvtConfigExtendedReport))
}

func TestHandleCmdConfigExtendedSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		value         any
		configureErr  error
		wantConfigure bool
		wantStatus    string
		wantError     string
	}{
		{
			name:          "accepted",
			value:         handlerConfig{Name: "n"},
			wantConfigure: true,
			wantStatus:    app.OperationStatusOK,
		},
		{
			name:          "rejected by the application",
			value:         handlerConfig{Name: "n"},
			configureErr:  errors.New("bad name"),
			wantConfigure: true,
			wantStatus:    app.OperationStatusError,
			wantError:     "configure the app err: bad name",
		},
		{
			name:       "unparsable payload",
			value:      []int{1},
			wantStatus: app.OperationStatusError,
			wantError:  "configure the app err: json:",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lc := lifecycle.New(nil)
			lc.SetConfigState(lifecycle.ConfigStateInProgress)

			application := &recordingApp{configureErr: tc.configureErr}
			handler := app.HandleCmdConfigExtendedSet(testDiagService, lc, func() *handlerConfig { return &handlerConfig{} }, application, nil)

			report := requireReply[*app.ConfigurationReport](t, handler.Handle(newRequest(t, app.CmdConfigExtendedSet, tc.value)), app.EvtAppConfigReport)

			assert.Equal(t, tc.wantStatus, report.OpStatus)
			assert.Contains(t, report.OpError, tc.wantError)
			assert.Equal(t, string(lifecycle.ConfigStateInProgress), report.AppState.Config)

			if tc.wantConfigure {
				assert.Equal(t, []any{&handlerConfig{Name: "n"}}, application.configured)
			} else {
				assert.Empty(t, application.configured)
			}
		})
	}
}

func TestHandleCmdAppUninstall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		exclude      func() error
		uninstallErr error
		wantStatus   string
	}{
		{name: "exclusion fails, uninstall still runs", exclude: func() error { return errors.New("exclude") }, wantStatus: app.OperationStatusOK},
		{name: "nil exclusion is skipped", wantStatus: app.OperationStatusOK},
		{name: "uninstall fails", exclude: func() error { return nil }, uninstallErr: errors.New("boom"), wantStatus: app.OperationStatusError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			application := &recordingApp{uninstallErr: tc.uninstallErr}
			handler := app.HandleCmdAppUninstall(testDiagService, lifecycle.New(nil), application, nil, tc.exclude)

			report := requireReply[*app.ConfigurationReport](t, handler.Handle(newRequest(t, app.CmdAppUninstall, nil)), app.EvtAppUninstallReport)

			assert.Equal(t, tc.wantStatus, report.OpStatus)
			assert.Equal(t, 1, application.uninstalls)
		})
	}
}

func TestHandleCmdAppUninstall_ExcludesThingsBeforeUninstall(t *testing.T) {
	t.Parallel()

	application := &recordingApp{}
	uninstallsAtExclude := -1

	handler := app.HandleCmdAppUninstall(testDiagService, lifecycle.New(nil), application, nil, func() error {
		uninstallsAtExclude = application.uninstalls

		return nil
	})

	handler.Handle(newRequest(t, app.CmdAppUninstall, nil))

	assert.Equal(t, 0, uninstallsAtExclude)
	assert.Equal(t, 1, application.uninstalls)
}

func TestHandleConfigActionCommand_PassesStringParameter(t *testing.T) {
	t.Parallel()

	var got []string

	handler := app.HandleConfigActionCommand(testDiagService, func(p string) *manifest.ButtonActionResponse {
		got = append(got, p)

		return &manifest.ButtonActionResponse{Operation: "cmd.app.custom", OperationStatus: "ok"}
	}, nil)

	reply := handler.Handle(newRequest(t, "cmd.app.custom", "device-1"))
	requireReply[*manifest.ButtonActionResponse](t, reply, app.EvtAppConfigActionReport)

	handler.Handle(newRequest(t, "cmd.app.custom", map[string]int{"a": 1}))

	assert.Equal(t, []string{"device-1", ""}, got)
}

func TestMutatingHandlers_RejectedWhileLocked(t *testing.T) {
	t.Parallel()

	lc := lifecycle.New(nil)
	locker := router.NewMessageHandlerLocker()
	application := &recordingApp{}
	factory := func() *handlerConfig { return &handlerConfig{} }

	require.True(t, locker.Lock())

	handlers := map[string]router.MessageHandler{
		app.CmdConfigExtendedSet: app.HandleCmdConfigExtendedSet(testDiagService, lc, factory, application, locker),
		app.CmdAppUninstall:      app.HandleCmdAppUninstall(testDiagService, lc, application, locker, nil),
		app.CmdAppReset: app.HandleConfigActionCommand(testDiagService, func(string) *manifest.ButtonActionResponse {
			_ = application.Reset()

			return &manifest.ButtonActionResponse{}
		}, locker),
		app.CmdAuthLogout:    app.HandleCmdAuthLogout(testDiagService, lc, locker, application),
		app.CmdAuthSetTokens: app.HandleCmdAuthSetTokens(testDiagService, lc, locker, application),
	}

	for msgType, handler := range handlers {
		requireErrorReport(t, handler.Handle(newRequest(t, msgType, handlerConfig{})), "another operation is already running")
	}

	assert.Empty(t, application.configured)
	assert.Zero(t, application.uninstalls)
	assert.Zero(t, application.resets)
	assert.Nil(t, application.authorizeArg)

	requireReply[*lifecycle.AppStateT](t, app.HandleCmdAppGetState(testDiagService, lc).Handle(newRequest(t, app.CmdAppGetState, nil)), app.EvtAppStateReport)

	locker.Unlock()

	handlers[app.CmdConfigExtendedSet].Handle(newRequest(t, app.CmdConfigExtendedSet, handlerConfig{}))
	handlers[app.CmdConfigExtendedSet].Handle(newRequest(t, app.CmdConfigExtendedSet, handlerConfig{}))

	assert.Len(t, application.configured, 2, "the handler must release the shared lock after each message")
}

func TestHandleCmdAuthSetTokens(t *testing.T) {
	t.Parallel()

	t.Run("authorize fails", func(t *testing.T) {
		t.Parallel()

		lc := lifecycle.New(nil)
		lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)

		application := &recordingApp{authorizeErr: errors.New("boom")}
		handler := app.HandleCmdAuthSetTokens(testDiagService, lc, nil, application)

		report := requireReply[*app.AuthenticationReport](t,
			handler.Handle(newRequest(t, app.CmdAuthSetTokens, auth.OAuth2TokenResponse{AccessToken: "at"})), app.EvtAuthStatusReport)

		assert.Equal(t, "failed to authorize", report.ErrorText)
		assert.Equal(t, string(lifecycle.AuthStateNotAuthenticated), report.Status)
		assert.Empty(t, report.Errors)
		require.NotNil(t, application.authorizeArg)
		assert.Equal(t, "at", application.authorizeArg.AccessToken)
	})

	t.Run("invalid payload", func(t *testing.T) {
		t.Parallel()

		application := &recordingApp{}
		handler := app.HandleCmdAuthSetTokens(testDiagService, lifecycle.New(nil), nil, application)

		requireErrorReport(t, handler.Handle(newRequest(t, app.CmdAuthSetTokens, "token")), "provided authorization credentials have an invalid format")
		assert.Nil(t, application.authorizeArg)
	})
}

func TestHandleCmdAuthLogout_ReportsStateAfterLogout(t *testing.T) {
	t.Parallel()

	lc := lifecycle.New(nil)
	lc.SetAuthState(lifecycle.AuthStateAuthenticated)

	handler := app.HandleCmdAuthLogout(testDiagService, lc, nil, logoutFunc(func() error {
		lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)

		return errors.New("partial")
	}))

	report := requireReply[*app.AuthenticationReport](t, handler.Handle(newRequest(t, app.CmdAuthLogout, nil)), app.EvtAuthStatusReport)

	assert.Equal(t, string(lifecycle.AuthStateNotAuthenticated), report.Status)
	assert.Equal(t, "failed to logout", report.ErrorText)
	assert.Empty(t, report.Errors)
}

func TestHandleCmdAuthLogin_InvalidCredentials(t *testing.T) {
	t.Parallel()

	handler := app.HandleCmdAuthLogin(testDiagService, lifecycle.New(nil), nil, stubDualApp{})

	requireErrorReport(t, handler.Handle(newRequest(t, app.CmdAuthLogin, []int{1})), "provided login credentials have an invalid format")
}

type logoutFunc func() error

func (f logoutFunc) Logout() error { return f() }
