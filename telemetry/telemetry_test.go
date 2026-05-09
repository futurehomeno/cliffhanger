package telemetry_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/cliffhanger/telemetry"
	"github.com/futurehomeno/cliffhanger/telemetry/types"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

const (
	testSource           = "core-energy-guard"
	appTopic             = "pt:j1/mt:cmd/rt:app/rn:" + string(telemetry.Service) + "/ad:1"
	appReportTopic       = "pt:j1/mt:evt/rt:app/rn:" + string(telemetry.Service) + "/ad:1"
	cmdSetEnabledType    = "cmd.config.set_" + telemetry.SettingEnabled
	cmdGetEnabledType    = "cmd.config.get_" + telemetry.SettingEnabled
	evtEnabledReport     = "evt.config." + telemetry.SettingEnabled + "_report"
	cmdSetValidityType   = "cmd.config.set_" + telemetry.SettingValidity
	cmdGetValidityType   = "cmd.config.get_" + telemetry.SettingValidity
	evtValidityReport    = "evt.config." + telemetry.SettingValidity + "_report"
	cmdSetSuppressedType = "cmd.config.set_" + telemetry.SettingSuppressed
	cmdGetSuppressedType = "cmd.config.get_" + telemetry.SettingSuppressed
	evtSuppressedReport  = "evt.config." + telemetry.SettingSuppressed + "_report"
)

func TestNew_RejectsInvalidInput(t *testing.T) {
	t.Parallel()

	t.Run("nil mqtt", func(t *testing.T) {
		t.Parallel()

		tel, err := telemetry.New(nil, testSource, telemetry.NewMemoryStore(true))

		require.Error(t, err)
		assert.Nil(t, tel)
	})

	t.Run("empty source", func(t *testing.T) {
		t.Parallel()

		tel, err := telemetry.New(&fimpgo.MqttTransport{}, "", telemetry.NewMemoryStore(true))

		require.Error(t, err)
		assert.Nil(t, tel)
	})

	t.Run("nil store", func(t *testing.T) {
		t.Parallel()

		tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, nil)

		require.Error(t, err)
		assert.Nil(t, tel)
	})

	t.Run("memory store with enabled=false boots disabled", func(t *testing.T) {
		t.Parallel()

		tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, telemetry.NewMemoryStore(false), telemetry.WithoutCloudPull())

		require.NoError(t, err)
		assert.False(t, tel.IsEnabled())
	})
}

func TestReporter(t *testing.T) { //nolint:paralleltest
	var tel telemetry.Telemetry

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "Telemetry",
				Setup: suite.BaseSetup(func(t *testing.T, mqtt *fimpgo.MqttTransport) (routing []*router.Routing, tasks []*task.Task, mocks []suite.Mock) {
					t.Helper()

					var err error

					tel, err = telemetry.New(mqtt, testSource, telemetry.NewMemoryStore(true), telemetry.WithoutCloudPull())
					require.NoError(t, err)

					return telemetry.RoutingForTelemetry(telemetry.Service, tel), nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Name: "Emit publishes the event",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								telemetry.Emit(tel, "energy_limit_exceeded", "max_guard", map[string]any{
									"hourly_energy_limit": 15.0,
								})
							},
						},
						Expectations: []*suite.Expectation{
							suite.NewExpectation(
								router.ForTopic(telemetry.Topic),
								router.ForType(telemetry.MessageType),
								router.ForService(telemetry.Service),
								router.MessageVoterFn(func(msg *fimpgo.Message) bool {
									if string(msg.Payload.Source) != testSource {
										return false
									}

									var got telemetry.Event
									if err := msg.Payload.GetObjectValue(&got); err != nil {
										return false
									}

									want := telemetry.Event{
										Event:  "energy_limit_exceeded",
										Domain: "max_guard",
										Data: map[string]any{
											"hourly_energy_limit": 15.0,
										},
									}

									return reflect.DeepEqual(got, want)
								}),
							),
						},
					},
					{
						Name: "Emit with empty event name does not publish",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								telemetry.Emit(tel, "", "auth", map[string]any{"x": 1})
							},
						},
						// No Expectations: empty event name is rejected by publish and Emit logs a warning.
					},
					{
						Name: "SetEvtTopic redirects publishes",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								tel.SetEvtTopic("pt:j1/mt:evt/rt:app/rn:custom/ad:1")

								telemetry.Emit(tel, "app_started", "", nil)
							},
						},
						Expectations: []*suite.Expectation{
							suite.NewExpectation(
								router.ForTopic("pt:j1/mt:evt/rt:app/rn:custom/ad:1"),
								router.ForType(telemetry.MessageType),
							),
						},
					},
					{
						Name: "SetEvtTopic with empty restores default",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								tel.SetEvtTopic("")

								telemetry.Emit(tel, "app_started", "", nil)
							},
						},
						Expectations: []*suite.Expectation{
							suite.NewExpectation(
								router.ForTopic(telemetry.DefaultTelemetryEvtTopic),
								router.ForType(telemetry.MessageType),
							),
						},
					},
					{
						Name: "Enable(false) silences Emit",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								require.NoError(t, tel.Enable(false))
								assert.False(t, tel.IsEnabled())

								telemetry.Emit(tel, "should_not_publish", "", nil)
							},
						},
						// No Expectations: nothing should be published.
					},
					{
						Name: "Enable(false) also drops EmitRequired",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								assert.False(t, tel.IsEnabled())

								telemetry.EmitRequired(tel, "should_not_publish", "", nil)
							},
						},
						// No Expectations: global Enabled=false drops everything.
					},
					{
						Name: "Enable(true) restores Emit",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								require.NoError(t, tel.Enable(true))
								assert.True(t, tel.IsEnabled())

								telemetry.Emit(tel, "after_enable", "", nil)
							},
						},
						Expectations: []*suite.Expectation{
							suite.NewExpectation(
								router.ForTopic(telemetry.Topic),
								router.ForType(telemetry.MessageType),
								router.MessageVoterFn(func(msg *fimpgo.Message) bool {
									var got telemetry.Event
									if err := msg.Payload.GetObjectValue(&got); err != nil {
										return false
									}

									return got.Event == "after_enable"
								}),
							),
						},
					},
					{
						Name:    "cmd.config.set_telemetry_enabled = false disables the telemetry",
						Command: suite.BoolMessage(appTopic, cmdSetEnabledType, telemetry.Service, false),
						Expectations: []*suite.Expectation{
							suite.ExpectBool(appReportTopic, evtEnabledReport, telemetry.Service, false),
						},
					},
					{
						Name:    "cmd.config.get_telemetry_enabled reports current state",
						Command: suite.NullMessage(appTopic, cmdGetEnabledType, telemetry.Service),
						Expectations: []*suite.Expectation{
							suite.ExpectBool(appReportTopic, evtEnabledReport, telemetry.Service, false),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								assert.False(t, tel.IsEnabled())
							},
						},
					},
					{
						Name:    "cmd.config.set_telemetry_enabled = true re-enables the telemetry",
						Command: suite.BoolMessage(appTopic, cmdSetEnabledType, telemetry.Service, true),
						Expectations: []*suite.Expectation{
							suite.ExpectBool(appReportTopic, evtEnabledReport, telemetry.Service, true),
						},
					},
					{
						Name: "Validity defaults to 30 days",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								assert.Equal(t, types.DefaultTelemetryValidity, tel.Validity())
							},
						},
					},
					{
						Name: "SetValidity rejects non-positive values",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								assert.Error(t, tel.SetValidity(0))
								assert.Error(t, tel.SetValidity(-time.Second))
							},
						},
					},
					{
						Name:    "cmd.config.set_telemetry_validity updates the window",
						Command: suite.StringMessage(appTopic, cmdSetValidityType, telemetry.Service, "1h"),
						Expectations: []*suite.Expectation{
							suite.ExpectString(appReportTopic, evtValidityReport, telemetry.Service, "1h"),
						},
					},
					{
						Name:    "cmd.config.get_telemetry_validity reports current window",
						Command: suite.NullMessage(appTopic, cmdGetValidityType, telemetry.Service),
						Expectations: []*suite.Expectation{
							suite.ExpectString(appReportTopic, evtValidityReport, telemetry.Service, "1h0m0s"),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								assert.Equal(t, time.Hour, tel.Validity())
							},
						},
					},
					{
						Name: "SetValidity below elapsed auto-disables immediately",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								require.NoError(t, tel.Enable(true))
								time.Sleep(20 * time.Millisecond)

								require.NoError(t, tel.SetValidity(time.Millisecond))
								assert.False(t, tel.IsEnabled())

								telemetry.Emit(tel, "should_not_publish", "", nil)
							},
						},
						// No Expectations: telemetry is disabled.
					},
					{
						Name: "Validity window expires and auto-disables",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								require.NoError(t, tel.SetValidity(50*time.Millisecond))
								require.NoError(t, tel.Enable(true))

								time.Sleep(150 * time.Millisecond)

								assert.False(t, tel.IsEnabled())

								telemetry.Emit(tel, "should_not_publish", "", nil)
							},
						},
						// No Expectations: timer disables the tel.
					},
					{
						Name: "Re-enable for suppressed-list tests",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								require.NoError(t, tel.SetValidity(types.DefaultTelemetryValidity))
								require.NoError(t, tel.Enable(true))
							},
						},
					},
					{
						Name: "SetSuppressedDomains adds a domain that drops Emit",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								require.NoError(t, tel.SetSuppressedDomains([]string{"noisy"}))
								assert.Equal(t, []string{"noisy"}, tel.SuppressedDomains())

								telemetry.Emit(tel, "should_not_publish", "noisy", nil)
							},
						},
						// No Expectations: domain is suppressed.
					},
					{
						Name: "EmitRequired publishes for a suppressed domain",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								assert.Equal(t, []string{"noisy"}, tel.SuppressedDomains())

								telemetry.EmitRequired(tel, "critical_event", "noisy", nil)
							},
						},
						Expectations: []*suite.Expectation{
							suite.NewExpectation(
								router.ForTopic(telemetry.Topic),
								router.ForType(telemetry.MessageType),
								router.MessageVoterFn(func(msg *fimpgo.Message) bool {
									var got telemetry.Event
									if err := msg.Payload.GetObjectValue(&got); err != nil {
										return false
									}

									return got.Event == "critical_event"
								}),
							),
						},
					},
					{
						Name: "EmitRequired with empty event name does not publish",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								telemetry.EmitRequired(tel, "", "noisy", nil)
							},
						},
						// No Expectations: empty event name is rejected by publish.
					},
					{
						Name: "SetSuppressedDomains(nil) clears the suppression",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								require.NoError(t, tel.SetSuppressedDomains(nil))
								assert.Empty(t, tel.SuppressedDomains())

								telemetry.Emit(tel, "after_unsuppress", "noisy", nil)
							},
						},
						Expectations: []*suite.Expectation{
							suite.NewExpectation(
								router.ForTopic(telemetry.Topic),
								router.ForType(telemetry.MessageType),
								router.MessageVoterFn(func(msg *fimpgo.Message) bool {
									var got telemetry.Event
									if err := msg.Payload.GetObjectValue(&got); err != nil {
										return false
									}

									return got.Event == "after_unsuppress"
								}),
							),
						},
					},
					{
						Name: "SuppressedDomains returns a defensive copy",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								require.NoError(t, tel.SetSuppressedDomains([]string{"a", "b"}))

								got := tel.SuppressedDomains()
								require.Len(t, got, 2)

								got[0] = "mutated"

								assert.Equal(t, []string{"a", "b"}, tel.SuppressedDomains(), "mutation must not affect store")
							},
						},
					},
					{
						Name:    "cmd.config.set_telemetry_suppressed replaces the suppression list",
						Command: suite.StringArrayMessage(appTopic, cmdSetSuppressedType, telemetry.Service, []string{"max_guard", "discovery"}),
						Expectations: []*suite.Expectation{
							suite.NewExpectation(
								router.ForTopic(appReportTopic),
								router.ForType(evtSuppressedReport),
								router.ForService(telemetry.Service),
							),
						},
					},
					{
						Name:    "cmd.config.get_telemetry_suppressed reports the current list",
						Command: suite.NullMessage(appTopic, cmdGetSuppressedType, telemetry.Service),
						Expectations: []*suite.Expectation{
							suite.NewExpectation(
								router.ForTopic(appReportTopic),
								router.ForType(evtSuppressedReport),
								router.ForService(telemetry.Service),
								router.MessageVoterFn(func(msg *fimpgo.Message) bool {
									got, err := msg.Payload.GetStrArrayValue()
									if err != nil {
										return false
									}

									return reflect.DeepEqual(got, []string{"max_guard", "discovery"})
								}),
							),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

// stubStore exposes the persisted *types.TelemetryConfig pointer for
// white-box tests of the constructor's restart-resume logic.
type stubStore struct {
	cfg       *types.TelemetryConfig
	saveCalls int
}

func TestNew_ResumesValidityWindowAcrossRestart(t *testing.T) {
	t.Parallel()

	t.Run("mid-window: resumes with remaining time", func(t *testing.T) {
		t.Parallel()

		store := &stubStore{
			cfg: &types.TelemetryConfig{
				Enabled:   true,
				EnabledAt: time.Now().Add(-10 * time.Minute),
				Validity:  time.Hour,
			},
		}

		tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store, telemetry.WithoutCloudPull())
		require.NoError(t, err)

		assert.True(t, tel.IsEnabled())
		assert.Equal(t, time.Hour, tel.Validity())
		assert.Equal(t, 0, store.saveCalls, "must not re-persist on resume")
	})

	t.Run("already expired: auto-disables and persists", func(t *testing.T) {
		t.Parallel()

		store := &stubStore{
			cfg: &types.TelemetryConfig{
				Enabled:   true,
				EnabledAt: time.Now().Add(-40 * 24 * time.Hour),
				Validity:  30 * 24 * time.Hour,
			},
		}

		tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store, telemetry.WithoutCloudPull())
		require.NoError(t, err)

		assert.False(t, tel.IsEnabled())
		require.NotNil(t, store.cfg)
		assert.False(t, store.cfg.Enabled)
		assert.True(t, store.cfg.EnabledAt.IsZero())
	})

	t.Run("enabled with zero enabledAt: stamps fresh window", func(t *testing.T) {
		t.Parallel()

		store := &stubStore{
			cfg: &types.TelemetryConfig{
				Enabled:  true,
				Validity: time.Hour,
			},
		}

		before := time.Now()

		tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store, telemetry.WithoutCloudPull())
		require.NoError(t, err)

		assert.True(t, tel.IsEnabled())
		require.NotNil(t, store.cfg)
		assert.False(t, store.cfg.EnabledAt.IsZero())
		assert.True(t, !store.cfg.EnabledAt.Before(before))
	})
}

// failStore returns the configured error from SetTelemetry and never
// mutates its fields - lets tests assert that a setter failure leaves
// the store untouched.
type failStore struct {
	cfg    *types.TelemetryConfig
	setErr error
}

func TestEnable_SaveFailure_LeavesStateUnchanged(t *testing.T) {
	t.Parallel()

	store := &failStore{
		cfg: &types.TelemetryConfig{
			Enabled:  false,
			Validity: types.DefaultTelemetryValidity,
		},
	}

	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store, telemetry.WithoutCloudPull())
	require.NoError(t, err)
	assert.False(t, tel.IsEnabled())

	store.setErr = errors.New("disk full")
	err = tel.Enable(true)
	require.Error(t, err)

	assert.False(t, tel.IsEnabled(), "Enable must not flip in-memory state on setter failure")
	assert.False(t, store.cfg.Enabled, "Store must not be mutated on setter failure")
}

func TestSetValidity_SaveFailure_LeavesStateUnchanged(t *testing.T) {
	t.Parallel()

	store := &failStore{
		cfg: &types.TelemetryConfig{
			Enabled:  true,
			Validity: types.DefaultTelemetryValidity,
		},
	}

	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store, telemetry.WithoutCloudPull())
	require.NoError(t, err)

	store.setErr = errors.New("disk full")
	err = tel.SetValidity(time.Hour)
	require.Error(t, err)

	assert.Equal(t, types.DefaultTelemetryValidity, tel.Validity(), "SetValidity must not change state on setter failure")
}

func TestSetValidity_BelowElapsed_AutoDisables(t *testing.T) {
	t.Parallel()

	store := &stubStore{
		cfg: &types.TelemetryConfig{
			Enabled:   true,
			EnabledAt: time.Now().Add(-10 * time.Minute),
			Validity:  time.Hour,
		},
	}

	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store, telemetry.WithoutCloudPull())
	require.NoError(t, err)
	assert.True(t, tel.IsEnabled())

	require.NoError(t, tel.SetValidity(time.Millisecond))

	assert.False(t, tel.IsEnabled(), "should auto-disable")
	require.NotNil(t, store.cfg)
	assert.False(t, store.cfg.Enabled, "persisted state must be disabled")
	assert.True(t, store.cfg.EnabledAt.IsZero(), "persisted enabledAt must be cleared")
}

func TestSetSuppressedDomains_SaveFailure_LeavesStateUnchanged(t *testing.T) {
	t.Parallel()

	store := &failStore{
		cfg: &types.TelemetryConfig{
			Enabled:  true,
			Validity: types.DefaultTelemetryValidity,
		},
	}

	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store, telemetry.WithoutCloudPull())
	require.NoError(t, err)
	assert.Empty(t, tel.SuppressedDomains())

	store.setErr = errors.New("disk full")
	err = tel.SetSuppressedDomains([]string{"x"})
	require.Error(t, err)

	assert.Empty(t, tel.SuppressedDomains(), "SetSuppressedDomains must not change state on setter failure")
}

func TestEnable_PreservesSuppressedDomainsInStore(t *testing.T) {
	t.Parallel()

	store := &stubStore{
		cfg: &types.TelemetryConfig{
			Enabled:           true,
			Validity:          types.DefaultTelemetryValidity,
			SuppressedDomains: []string{"keep-me"},
		},
	}

	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store, telemetry.WithoutCloudPull())
	require.NoError(t, err)
	assert.Equal(t, []string{"keep-me"}, tel.SuppressedDomains())

	require.NoError(t, tel.Enable(true))
	assert.Equal(t, []string{"keep-me"}, store.cfg.SuppressedDomains, "Enable must not touch the persisted suppressed list")
}

func TestEmit_NilTelemetry(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		telemetry.Emit(nil, "test_event", "domain", map[string]any{"key": "val"})
	})
}

func TestEmitRequired_NilTelemetry(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		telemetry.EmitRequired(nil, "test_event", "domain", map[string]any{"key": "val"})
	})
}
