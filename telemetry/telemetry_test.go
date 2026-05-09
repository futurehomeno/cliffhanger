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

func boolPtr(b bool) *bool { return &b }

// stubStore exposes raw fields for white-box testing of New's restart-resume
// logic. Unlike NewMemoryStore, it lets tests seed non-zero values.
type stubStore struct {
	enabled    *bool
	enabledAt  time.Time
	validity   time.Duration
	suppressed *bool
	saveCalls  int
}

func (s *stubStore) Enabled() *bool { return s.enabled }

func (s *stubStore) SetEnabled(v *bool) error {
	s.enabled = v
	s.saveCalls++

	return nil
}

func (s *stubStore) EnabledAt() time.Time { return s.enabledAt }

func (s *stubStore) Validity() time.Duration { return s.validity }

func (s *stubStore) SetValidity(d time.Duration) error {
	s.validity = d
	s.saveCalls++

	return nil
}

func (s *stubStore) Suppressed() *bool { return s.suppressed }

func (s *stubStore) SetSuppressed(v *bool) error {
	s.suppressed = v
	s.saveCalls++

	return nil
}

func TestNew_ResumesValidityWindowAcrossRestart(t *testing.T) {
	t.Parallel()

	t.Run("mid-window: resumes with remaining time", func(t *testing.T) {
		t.Parallel()

		store := &stubStore{
			enabled:   boolPtr(true),
			enabledAt: time.Now().Add(-10 * time.Minute),
			validity:  time.Hour,
		}

		tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
		require.NoError(t, err)

		assert.True(t, tel.IsEnabled())
		assert.Equal(t, time.Hour, tel.Validity())
		assert.Equal(t, 0, store.saveCalls, "must not re-persist on resume")
	})

	t.Run("already expired: auto-disables and persists", func(t *testing.T) {
		t.Parallel()

		store := &stubStore{
			enabled:   boolPtr(true),
			enabledAt: time.Now().Add(-40 * 24 * time.Hour),
			validity:  30 * 24 * time.Hour,
		}

		tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
		require.NoError(t, err)

		assert.False(t, tel.IsEnabled())
		require.NotNil(t, store.enabled)
		assert.False(t, *store.enabled)
		assert.True(t, store.enabledAt.IsZero())
	})

	t.Run("enabled with zero enabledAt: stamps fresh window", func(t *testing.T) {
		t.Parallel()

		store := &stubStore{
			enabled:  boolPtr(true),
			validity: time.Hour,
		}

		before := time.Now()

		tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
		require.NoError(t, err)

		assert.True(t, tel.IsEnabled())
		assert.False(t, store.enabledAt.IsZero())
		assert.True(t, !store.enabledAt.Before(before))
	})
}

// failStore returns the configured error from every setter and never mutates
// its fields - lets tests assert that a setter failure leaves the store
// untouched.
type failStore struct {
	enabled    *bool
	enabledAt  time.Time
	validity   time.Duration
	suppressed *bool
	setErr     error
}

func (s *failStore) Enabled() *bool { return s.enabled }

func (s *failStore) SetEnabled(v *bool) error {
	if s.setErr != nil {
		return s.setErr
	}

	s.enabled = v

	return nil
}

func (s *failStore) EnabledAt() time.Time { return s.enabledAt }

func (s *failStore) SetEnabledAt(t time.Time) error {
	if s.setErr != nil {
		return s.setErr
	}

	s.enabledAt = t

	return nil
}

func (s *failStore) Validity() time.Duration { return s.validity }

func (s *failStore) SetValidity(d time.Duration) error {
	if s.setErr != nil {
		return s.setErr
	}

	s.validity = d

	return nil
}

func (s *failStore) Suppressed() *bool { return s.suppressed }

func (s *failStore) SetSuppressed(v *bool) error {
	if s.setErr != nil {
		return s.setErr
	}

	s.suppressed = v

	return nil
}

func TestEnable_SaveFailure_LeavesStateUnchanged(t *testing.T) {
	t.Parallel()

	store := &failStore{
		enabled:  boolPtr(false),
		validity: telemetry.DefaultValidity,
	}

	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
	require.NoError(t, err)
	assert.False(t, tel.IsEnabled())

	// Inject failure and try to enable.
	store.setErr = errors.New("disk full")
	err = tel.Enable(true)
	require.Error(t, err)

	// In-memory state must remain disabled.
	assert.False(t, tel.IsEnabled(), "Enable must not change in-memory state on setter failure")
	require.NotNil(t, store.enabled)
	assert.False(t, *store.enabled, "Store must not be mutated on setter failure")
}

func TestSetValidity_SaveFailure_LeavesStateUnchanged(t *testing.T) {
	t.Parallel()

	store := &failStore{
		enabled:  boolPtr(true),
		validity: telemetry.DefaultValidity,
	}

	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
	require.NoError(t, err)

	store.setErr = errors.New("disk full")
	err = tel.SetValidity(time.Hour)
	require.Error(t, err)

	assert.Equal(t, telemetry.DefaultValidity, tel.Validity(), "SetValidity must not change in-memory state on setter failure")
}

func TestSetValidity_BelowElapsed_AutoDisables(t *testing.T) {
	t.Parallel()

	store := &stubStore{
		enabled:   boolPtr(true),
		enabledAt: time.Now().Add(-10 * time.Minute),
		validity:  time.Hour,
	}

	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
	require.NoError(t, err)
	assert.True(t, tel.IsEnabled())

	require.NoError(t, tel.SetValidity(time.Millisecond))

	assert.False(t, tel.IsEnabled(), "should auto-disable")
	require.NotNil(t, store.enabled)
	assert.False(t, *store.enabled, "persisted state must be disabled")
	assert.True(t, store.enabledAt.IsZero(), "persisted enabledAt must be cleared")
}

func TestSetSuppressed_SaveFailure_LeavesStateUnchanged(t *testing.T) {
	t.Parallel()

	store := &failStore{
		enabled:  boolPtr(true),
		validity: telemetry.DefaultValidity,
	}

	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
	require.NoError(t, err)
	assert.False(t, tel.IsSuppressed())

	store.setErr = errors.New("disk full")
	err = tel.SetSuppressed(true)
	require.Error(t, err)

	assert.False(t, tel.IsSuppressed(), "SetSuppressed must not change in-memory state on setter failure")
}

func TestEnable_PreservesSuppressedInStore(t *testing.T) {
	t.Parallel()

	store := &stubStore{
		enabled:    boolPtr(true),
		suppressed: boolPtr(true),
		validity:   telemetry.DefaultValidity,
	}

	tel, err := telemetry.New(&fimpgo.MqttTransport{}, testSource, store)
	require.NoError(t, err)
	assert.True(t, tel.IsSuppressed())

	require.NoError(t, tel.Enable(true))
	require.NotNil(t, store.suppressed)
	assert.True(t, *store.suppressed, "Enable must not touch the persisted suppressed flag")
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
