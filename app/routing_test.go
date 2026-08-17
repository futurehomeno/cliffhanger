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

type stubConfig struct{}

// stubAuthApp implements app.App + app.AuthorizableApp (OAuth2). Handlers are never invoked in
// these tests — only RouteApp's build-time auth-state default is exercised.
type stubAuthApp struct{}

func (stubAuthApp) GetManifest() (*manifest.Manifest, error)  { return &manifest.Manifest{}, nil }
func (stubAuthApp) Configure(any) error                       { return nil }
func (stubAuthApp) Uninstall() error                          { return nil }
func (stubAuthApp) Logout() error                             { return nil }
func (stubAuthApp) Authorize(*auth.OAuth2TokenResponse) error { return nil }
func (stubAuthApp) ErrorsReport() ([]string, error)           { return nil, nil }

// stubLoginApp implements app.App + app.LogginableApp (password/token) — NOT AuthorizableApp.
type stubLoginApp struct{}

func (stubLoginApp) GetManifest() (*manifest.Manifest, error) { return &manifest.Manifest{}, nil }
func (stubLoginApp) Configure(any) error                      { return nil }
func (stubLoginApp) Uninstall() error                         { return nil }
func (stubLoginApp) Logout() error                            { return nil }
func (stubLoginApp) Login(*app.LoginCredentials) error        { return nil }
func (stubLoginApp) ErrorsReport() ([]string, error)          { return nil, nil }

func TestRouteApp_DefaultsAuthorizableAppToNotAuthenticated(t *testing.T) {
	t.Parallel()

	build := func(t *testing.T, application app.App, lc *lifecycle.Lifecycle) {
		t.Helper()

		st := storage.New(&stubConfig{}, t.TempDir(), "config.json")

		app.RouteApp[*stubConfig](
			testDiagService, lc, st,
			func() *stubConfig { return &stubConfig{} },
			router.NewMessageHandlerLocker(),
			application,
			func() error { return nil },
		)
	}

	t.Run("authorizable app left undecided defaults to not authenticated", func(t *testing.T) {
		t.Parallel()

		lc := lifecycle.New(nil) // auth starts at NA
		build(t, stubAuthApp{}, lc)

		assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
	})

	t.Run("authorizable app already authenticated is left untouched", func(t *testing.T) {
		t.Parallel()

		lc := lifecycle.New(nil)
		lc.SetAuthState(lifecycle.AuthStateAuthenticated) // adapter loaded creds in Initialize
		build(t, stubAuthApp{}, lc)

		assert.Equal(t, lifecycle.AuthStateAuthenticated, lc.AuthState())
	})

	t.Run("login-only app is not defaulted, the fallback is OAuth2-only", func(t *testing.T) {
		t.Parallel()

		lc := lifecycle.New(nil)
		build(t, stubLoginApp{}, lc)

		assert.Equal(t, lifecycle.AuthStateNA, lc.AuthState())
	})
}

// stubDualApp supports both credential login and OAuth2, as a real adapter offering two sign-in
// methods does. Both interfaces embed LogoutableApp.
type stubDualApp struct {
	stubLoginApp

	loginErr error
}

func (stubDualApp) Authorize(*auth.OAuth2TokenResponse) error { return nil }

func (s stubDualApp) Login(*app.LoginCredentials) error { return s.loginErr }

// TestRouteApp_RegistersLogoutOnce pins that an app implementing both authentication interfaces
// gets a single cmd.auth.logout routing. The router dispatches to every matching routing, so a
// duplicate made one command log out twice and publish two contradictory status reports.
func TestRouteApp_RegistersLogoutOnce(t *testing.T) {
	t.Parallel()

	route := func(application app.App) []*router.Routing {
		return app.RouteApp[*stubConfig](
			testDiagService, lifecycle.New(nil), storage.New(&stubConfig{}, t.TempDir(), "config.json"),
			func() *stubConfig { return &stubConfig{} },
			router.NewMessageHandlerLocker(),
			application,
			func() error { return nil },
		)
	}

	// The only routing the dual app adds over the login-only one is cmd.auth.set_tokens.
	assert.Len(t, route(stubDualApp{}), len(route(stubLoginApp{}))+1)
	assert.Len(t, route(stubAuthApp{}), len(route(stubLoginApp{})))
}

func newLoginRequest(t *testing.T) *fimpgo.Message {
	t.Helper()

	addr, err := fimpgo.NewAddressFromString("pt:j1/mt:cmd/rt:app/rn:" + testDiagService + "/ad:1")
	require.NoError(t, err)

	// Round-tripped through JSON so that the object value is available the way it is after an
	// actual MQTT delivery.
	body, err := fimpgo.NewObjectMessage(
		app.CmdAuthLogin, testDiagService, &app.LoginCredentials{Username: "u", Password: "p"}, nil, nil, nil,
	).SerializeToJson()
	require.NoError(t, err)

	payload, err := fimpgo.NewMessageFromBytes(body)
	require.NoError(t, err)

	return &fimpgo.Message{Topic: addr.Serialize(), Addr: addr, Payload: payload}
}

// TestHandleCmdAuthLogin_FHXErrorsField pins that the deprecated errors field marks only outcomes
// that actually failed. It used to be set for any state other than AUTHENTICATED, which reports a
// login that legitimately has not finished as a failure.
func TestHandleCmdAuthLogin_FHXErrorsField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		state    lifecycle.State
		loginErr error
		want     string
	}{
		{name: "authenticated", state: lifecycle.AuthStateAuthenticated},
		{name: "still in progress", state: lifecycle.AuthStateInProgress},
		{name: "rejected", state: lifecycle.AuthStateNotAuthenticated, want: "failed to login"},
		{name: "errored", state: lifecycle.AuthStateError, want: "failed to login"},
		{
			name:     "login returned an error",
			state:    lifecycle.AuthStateInProgress,
			loginErr: errors.New("boom"),
			want:     "failed to login",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			l := lifecycle.New(nil)
			l.SetAuthState(tc.state)

			handler := app.HandleCmdAuthLogin(
				testDiagService, l, router.NewMessageHandlerLocker(), stubDualApp{loginErr: tc.loginErr},
			)

			reply := handler.Handle(newLoginRequest(t))
			require.NotNil(t, reply)

			report, ok := reply.Payload.Value.(*app.AuthenticationReport)
			require.True(t, ok, "expected *app.AuthenticationReport payload, got %T", reply.Payload.Value)

			assert.Equal(t, tc.want, report.Errors)
			assert.Equal(t, string(tc.state), report.Status)
		})
	}
}

const testDiagService = "test_app"

// stubLogProvider is a test double for app.LogProvider.
type stubLogProvider struct {
	entries []string
	err     error
}

func (s *stubLogProvider) ErrorsReport() ([]string, error) {
	return s.entries, s.err
}

func newDiagRequest(t *testing.T) *fimpgo.Message {
	t.Helper()

	addr, err := fimpgo.NewAddressFromString("pt:j1/mt:cmd/rt:app/rn:" + testDiagService + "/ad:1")
	require.NoError(t, err)

	return &fimpgo.Message{
		Topic:   addr.Serialize(),
		Addr:    addr,
		Payload: fimpgo.NewNullMessage(app.CmdAppDiagGetReport, testDiagService, nil, nil, nil),
	}
}

func TestHandleCmdAppDiagGetReport_EmitsFullReport(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)

	logs := &stubLogProvider{entries: []string{"ERR one", "WARN two"}}

	handler := app.HandleCmdAppDiagGetReport(testDiagService, l, logs)

	reply := handler.Handle(newDiagRequest(t))

	require.NotNil(t, reply)
	require.NotNil(t, reply.Payload)

	assert.Equal(t, app.EvtAppDiagReport, reply.Payload.Interface)
	assert.Equal(t, "object", string(reply.Payload.ValueType))

	// The handler builds the payload in-memory, so the struct lives in
	// Payload.Value (no MQTT round-trip has serialized it into ValueObj yet).
	got, ok := reply.Payload.Value.(*app.DiagReport)
	require.True(t, ok, "expected *app.DiagReport payload, got %T", reply.Payload.Value)

	assert.GreaterOrEqual(t, got.Uptime, 0)
	assert.Equal(t, 0, got.RestartsCount)
	assert.Equal(t, []string{"ERR one", "WARN two"}, got.Errors)
}

func TestHandleCmdAppDiagGetReport_PropagatesLogProviderError(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)
	boom := errors.New("log read boom")
	logs := &stubLogProvider{err: boom}

	handler := app.HandleCmdAppDiagGetReport(testDiagService, l, logs)

	reply := handler.Handle(newDiagRequest(t))

	// On error the default handler emits an evt.error.report reply rather than
	// the diag report.
	require.NotNil(t, reply)
	require.NotNil(t, reply.Payload)
	assert.NotEqual(t, app.EvtAppDiagReport, reply.Payload.Interface, "must not emit diag report on error")
}
