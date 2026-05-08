package telemetry_test

import (
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/cliffhanger/telemetry"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

const (
	testSource            = "core-energy-guard"
	appTopic              = "pt:j1/mt:cmd/rt:app/rn:" + string(telemetry.Service) + "/ad:1"
	appReportTopic        = "pt:j1/mt:evt/rt:app/rn:" + string(telemetry.Service) + "/ad:1"
	cmdSetEnabledType     = "cmd.config.set_" + telemetry.SettingEnabled
	cmdGetEnabledType     = "cmd.config.get_" + telemetry.SettingEnabled
	evtEnabledReport      = "evt.config." + telemetry.SettingEnabled + "_report"
	cmdSetValidityType    = "cmd.config.set_" + telemetry.SettingValidity
	cmdGetValidityType    = "cmd.config.get_" + telemetry.SettingValidity
	evtValidityReport     = "evt.config." + telemetry.SettingValidity + "_report"
	cmdSetSuppressedType  = "cmd.config.set_" + telemetry.SettingSuppressed
	cmdGetSuppressedType  = "cmd.config.get_" + telemetry.SettingSuppressed
	evtSuppressedReport   = "evt.config." + telemetry.SettingSuppressed + "_report"
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

		tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, telemetry.NewMemoryStore(false))

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

					tel, err = telemetry.New(mqtt, testSource, telemetry.NewMemoryStore(true))
					require.NoError(t, err)

					return telemetry.RoutingForTelemetry(telemetry.Service, tel), nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Name: "Report publishes the event",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								err := tel.Report("energy_limit_exceeded", "max_guard", map[string]any{
									"hourly_energy_limit": 15.0,
								})

								assert.NoError(t, err)
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
						Name: "Report with empty event name returns error",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								err := tel.Report("", "auth", map[string]any{"x": 1})

								assert.Error(t, err)
							},
						},
					},
					{
						Name: "SetTargetTopic redirects publishes",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								tel.SetTargetTopic("pt:j1/mt:evt/rt:app/rn:custom/ad:1")

								err := tel.Report("app_started", "", nil)
								assert.NoError(t, err)
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
						Name: "SetTargetTopic with empty restores default",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								tel.SetTargetTopic("")

								err := tel.Report("app_started", "", nil)
								assert.NoError(t, err)
							},
						},
						Expectations: []*suite.Expectation{
							suite.NewExpectation(
								router.ForTopic(telemetry.Topic),
								router.ForType(telemetry.MessageType),
							),
						},
					},
					{
						Name: "Enable(false) silences Report",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								require.NoError(t, tel.Enable(false))
								assert.False(t, tel.IsEnabled())

								err := tel.Report("should_not_publish", "", nil)
								assert.NoError(t, err)
							},
						},
						// No Expectations: nothing should be published.
					},
					{
						Name: "Enable(true) restores Report",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								require.NoError(t, tel.Enable(true))
								assert.True(t, tel.IsEnabled())

								err := tel.Report("after_enable", "", nil)
								assert.NoError(t, err)
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

								assert.Equal(t, telemetry.DefaultValidity, tel.Validity())
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

								err := tel.Report("should_not_publish", "", nil)
								assert.NoError(t, err)
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

								err := tel.Report("should_not_publish", "", nil)
								assert.NoError(t, err)
							},
						},
						// No Expectations: timer disables the tel.
					},
					{
						Name: "Re-enable for suppressed tests",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								require.NoError(t, tel.SetValidity(telemetry.DefaultValidity))
								require.NoError(t, tel.Enable(true))
							},
						},
					},
					{
						Name: "SetSuppressed(true) silences Report",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								require.NoError(t, tel.SetSuppressed(true))
								assert.True(t, tel.IsSuppressed())

								err := tel.Report("should_not_publish", "", nil)
								assert.NoError(t, err)
							},
						},
						// No Expectations: suppressed.
					},
					{
						Name: "ReportRequired publishes when suppressed",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								assert.True(t, tel.IsSuppressed())

								err := tel.ReportRequired("critical_event", "health", nil)
								assert.NoError(t, err)
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
						Name: "ReportRequired with empty event name returns error",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								err := tel.ReportRequired("", "health", nil)
								assert.Error(t, err)
							},
						},
					},
					{
						Name: "SetSuppressed(false) restores Report",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								require.NoError(t, tel.SetSuppressed(false))
								assert.False(t, tel.IsSuppressed())

								err := tel.Report("after_unsuppress", "", nil)
								assert.NoError(t, err)
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
						Name: "ReportRequired publishes when disabled",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								require.NoError(t, tel.Enable(false))

								err := tel.ReportRequired("critical_while_disabled", "", nil)
								assert.NoError(t, err)
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

									return got.Event == "critical_while_disabled"
								}),
							),
						},
					},
					{
						Name: "Re-enable after ReportRequired disabled test",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								require.NoError(t, tel.Enable(true))
							},
						},
					},
					{
						Name:    "cmd.config.set_telemetry_suppressed = true suppresses the source",
						Command: suite.BoolMessage(appTopic, cmdSetSuppressedType, telemetry.Service, true),
						Expectations: []*suite.Expectation{
							suite.ExpectBool(appReportTopic, evtSuppressedReport, telemetry.Service, true),
						},
					},
					{
						Name:    "cmd.config.get_telemetry_suppressed reports current state",
						Command: suite.NullMessage(appTopic, cmdGetSuppressedType, telemetry.Service),
						Expectations: []*suite.Expectation{
							suite.ExpectBool(appReportTopic, evtSuppressedReport, telemetry.Service, true),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								assert.True(t, tel.IsSuppressed())
							},
						},
					},
					{
						Name:    "cmd.config.set_telemetry_suppressed = false restores the source",
						Command: suite.BoolMessage(appTopic, cmdSetSuppressedType, telemetry.Service, false),
						Expectations: []*suite.Expectation{
							suite.ExpectBool(appReportTopic, evtSuppressedReport, telemetry.Service, false),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

// stubStore exposes raw fields for white-box testing of New's restart-resume
// logic. Unlike NewMemoryStore, it lets tests seed non-zero state values.
type stubStore struct {
	state     telemetry.State
	saveCalls int
}

func (s *stubStore) Load() telemetry.State { return s.state }

func (s *stubStore) Save(st telemetry.State) error {
	s.state = st
	s.saveCalls++

	return nil
}

func TestNew_ResumesValidityWindowAcrossRestart(t *testing.T) {
	t.Parallel()

	t.Run("mid-window: resumes with remaining time", func(t *testing.T) {
		t.Parallel()

		store := &stubStore{state: telemetry.State{
			Enabled:   true,
			EnabledAt: time.Now().Add(-10 * time.Minute),
			Validity:  time.Hour,
		}}

		tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
		require.NoError(t, err)

		assert.True(t, tel.IsEnabled())
		assert.Equal(t, time.Hour, tel.Validity())
		assert.Equal(t, 0, store.saveCalls, "must not re-persist on resume")
	})

	t.Run("already expired: auto-disables and persists", func(t *testing.T) {
		t.Parallel()

		store := &stubStore{state: telemetry.State{
			Enabled:   true,
			EnabledAt: time.Now().Add(-40 * 24 * time.Hour),
			Validity:  30 * 24 * time.Hour,
		}}

		tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
		require.NoError(t, err)

		assert.False(t, tel.IsEnabled())
		assert.False(t, store.state.Enabled)
		assert.True(t, store.state.EnabledAt.IsZero())
	})

	t.Run("enabled with zero enabledAt: stamps fresh window", func(t *testing.T) {
		t.Parallel()

		store := &stubStore{state: telemetry.State{
			Enabled:  true,
			Validity: time.Hour,
		}}

		before := time.Now()

		tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
		require.NoError(t, err)

		assert.True(t, tel.IsEnabled())
		assert.False(t, store.state.EnabledAt.IsZero())
		assert.True(t, !store.state.EnabledAt.Before(before))
	})
}

func TestNewDefaultStore_RoundTrip(t *testing.T) {
	t.Parallel()

	d := &config.Default{}
	saves := 0
	store := telemetry.NewDefaultStore(func() *config.Default { return d }, func() error { saves++; return nil })

	// Defaults when nothing is persisted.
	st := store.Load()
	assert.True(t, st.Enabled, "unset enabled should default to true")
	assert.True(t, st.EnabledAt.IsZero())
	assert.Equal(t, telemetry.DefaultValidity, st.Validity)

	stamp := time.Date(2026, 4, 20, 12, 0, 0, 123_000_000, time.UTC)

	require.NoError(t, store.Save(telemetry.State{
		Enabled:   false,
		EnabledAt: stamp,
		Validity:  2 * time.Hour,
	}))

	// Load reflects the writes.
	st = store.Load()
	assert.False(t, st.Enabled)
	assert.Equal(t, 2*time.Hour, st.Validity)
	assert.True(t, st.EnabledAt.Equal(stamp))

	// Raw fields serialize the way the format contract says they should.
	assert.NotNil(t, d.TelemetryEnabled)
	assert.False(t, *d.TelemetryEnabled)
	assert.Equal(t, "2h0m0s", d.TelemetryValidity)
	assert.Equal(t, "2026-04-20T12:00:00.123Z", d.TelemetryEnabledAt)

	// Clearing enabledAt writes an empty string, not "0001-01-01..." -
	// a malformed RFC3339Nano in the file must round-trip to zero time.
	require.NoError(t, store.Save(telemetry.State{
		Enabled:  false,
		Validity: 2 * time.Hour,
	}))
	assert.Equal(t, "", d.TelemetryEnabledAt)

	st = store.Load()
	assert.True(t, st.EnabledAt.IsZero())

	assert.Equal(t, 2, saves, "each Save must call save exactly once")
}

// failStore lets tests inject Save failures.
type failStore struct {
	state   telemetry.State
	saveErr error
}

func (s *failStore) Load() telemetry.State { return s.state }

func (s *failStore) Save(st telemetry.State) error {
	if s.saveErr != nil {
		return s.saveErr
	}

	s.state = st

	return nil
}

func TestEnable_SaveFailure_LeavesStateUnchanged(t *testing.T) {
	t.Parallel()

	store := &failStore{state: telemetry.State{
		Enabled:  false,
		Validity: telemetry.DefaultValidity,
	}}

	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
	require.NoError(t, err)
	assert.False(t, tel.IsEnabled())

	// Inject failure and try to enable.
	store.saveErr = errors.New("disk full")
	err = tel.Enable(true)
	require.Error(t, err)

	// In-memory state must remain disabled.
	assert.False(t, tel.IsEnabled(), "Enable must not change in-memory state on Save failure")
	assert.False(t, store.state.Enabled, "Store must not be mutated on Save failure")
}

func TestSetValidity_SaveFailure_LeavesStateUnchanged(t *testing.T) {
	t.Parallel()

	store := &failStore{state: telemetry.State{
		Enabled:  true,
		Validity: telemetry.DefaultValidity,
	}}

	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
	require.NoError(t, err)

	store.saveErr = errors.New("disk full")
	err = tel.SetValidity(time.Hour)
	require.Error(t, err)

	assert.Equal(t, telemetry.DefaultValidity, tel.Validity(), "SetValidity must not change in-memory state on Save failure")
}

func TestSetValidity_BelowElapsed_SingleSave(t *testing.T) {
	t.Parallel()

	store := &stubStore{state: telemetry.State{
		Enabled:   true,
		EnabledAt: time.Now().Add(-10 * time.Minute),
		Validity:  time.Hour,
	}}

	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
	require.NoError(t, err)
	assert.True(t, tel.IsEnabled())

	store.saveCalls = 0

	require.NoError(t, tel.SetValidity(time.Millisecond))

	assert.False(t, tel.IsEnabled(), "should auto-disable")
	assert.Equal(t, 1, store.saveCalls, "SetValidity must perform exactly one Save")
	assert.False(t, store.state.Enabled, "persisted state must be disabled")
	assert.True(t, store.state.EnabledAt.IsZero(), "persisted enabledAt must be cleared")
}

func TestDefaultStore_SaveFailure_RestoresConfig(t *testing.T) {
	t.Parallel()

	enabled := true
	d := &config.Default{
		TelemetryEnabled:   &enabled,
		TelemetryEnabledAt: "2026-04-20T12:00:00Z",
		TelemetryValidity:  "1h0m0s",
	}

	fail := true
	store := telemetry.NewDefaultStore(
		func() *config.Default { return d },
		func() error {
			if fail {
				return errors.New("disk full")
			}

			return nil
		},
	)

	err := store.Save(telemetry.State{
		Enabled:  false,
		Validity: 2 * time.Hour,
	})
	require.Error(t, err)

	// Config must be rolled back to pre-Save state.
	assert.NotNil(t, d.TelemetryEnabled)
	assert.True(t, *d.TelemetryEnabled, "TelemetryEnabled must be restored")
	assert.Equal(t, "2026-04-20T12:00:00Z", d.TelemetryEnabledAt, "TelemetryEnabledAt must be restored")
	assert.Equal(t, "1h0m0s", d.TelemetryValidity, "TelemetryValidity must be restored")

	// Load must still return the original state.
	st := store.Load()
	assert.True(t, st.Enabled)
}

func TestSetSuppressed_SaveFailure_LeavesStateUnchanged(t *testing.T) {
	t.Parallel()

	store := &failStore{state: telemetry.State{
		Enabled:  true,
		Validity: telemetry.DefaultValidity,
	}}

	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
	require.NoError(t, err)
	assert.False(t, tel.IsSuppressed())

	store.saveErr = errors.New("disk full")
	err = tel.SetSuppressed(true)
	require.Error(t, err)

	assert.False(t, tel.IsSuppressed(), "SetSuppressed must not change in-memory state on Save failure")
}

func TestEnable_PreservesSuppressedInStore(t *testing.T) {
	t.Parallel()

	store := &stubStore{state: telemetry.State{
		Enabled:    true,
		Suppressed: true,
		Validity:   telemetry.DefaultValidity,
	}}

	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
	require.NoError(t, err)
	assert.True(t, tel.IsSuppressed())

	// Enable(true) must preserve the suppressed flag in the saved state.
	store.saveCalls = 0

	require.NoError(t, tel.Enable(true))
	assert.True(t, store.state.Suppressed, "Enable must preserve Suppressed in persisted state")
}

func TestNewDefaultStore_SuppressedRoundTrip(t *testing.T) {
	t.Parallel()

	d := &config.Default{}
	store := telemetry.NewDefaultStore(func() *config.Default { return d }, func() error { return nil })

	// Default: not suppressed.
	st := store.Load()
	assert.False(t, st.Suppressed, "unset suppressed should default to false")

	// Save with suppressed=true.
	require.NoError(t, store.Save(telemetry.State{
		Enabled:    true,
		Suppressed: true,
		Validity:   telemetry.DefaultValidity,
	}))

	st = store.Load()
	assert.True(t, st.Suppressed)
	assert.NotNil(t, d.TelemetrySuppressed)
	assert.True(t, *d.TelemetrySuppressed)

	// Save with suppressed=false.
	require.NoError(t, store.Save(telemetry.State{
		Enabled:  true,
		Validity: telemetry.DefaultValidity,
	}))

	st = store.Load()
	assert.False(t, st.Suppressed)
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

func TestNewLockedStore_DelegatesToInner(t *testing.T) {
	t.Parallel()

	inner := telemetry.NewMemoryStore(false)
	mu := &sync.RWMutex{}
	store := telemetry.NewLockedStore(inner, mu)

	st := store.Load()
	assert.False(t, st.Enabled)

	require.NoError(t, store.Save(telemetry.State{
		Enabled:  true,
		Validity: 48 * time.Hour,
	}))

	st = store.Load()
	assert.True(t, st.Enabled)
	assert.Equal(t, 48*time.Hour, st.Validity)
}

func TestNewLockedStore_PanicsOnNilStore(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		telemetry.NewLockedStore(nil, &sync.RWMutex{})
	})
}

func TestNewLockedStore_PanicsOnNilMutex(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		telemetry.NewLockedStore(telemetry.NewMemoryStore(false), nil)
	})
}
