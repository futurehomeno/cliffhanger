package telemetry_test

import (
	"reflect"
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
	testSource         = "core-energy-guard"
	appTopic           = "pt:j1/mt:cmd/rt:app/rn:" + string(telemetry.Service) + "/ad:1"
	appReportTopic     = "pt:j1/mt:evt/rt:app/rn:" + string(telemetry.Service) + "/ad:1"
	cmdSetEnabledType  = "cmd.config.set_" + telemetry.SettingEnabled
	cmdGetEnabledType  = "cmd.config.get_" + telemetry.SettingEnabled
	evtEnabledReport   = "evt.config." + telemetry.SettingEnabled + "_report"
	cmdSetValidityType = "cmd.config.set_" + telemetry.SettingValidity
	cmdGetValidityType = "cmd.config.get_" + telemetry.SettingValidity
	evtValidityReport  = "evt.config." + telemetry.SettingValidity + "_report"
)

func TestNew_RejectsInvalidInput(t *testing.T) {
	t.Parallel()

	t.Run("nil mqtt", func(t *testing.T) {
		t.Parallel()

		telemetry, err := telemetry.New(nil, testSource, telemetry.NewMemoryStore(true))

		require.Error(t, err)
		assert.Nil(t, telemetry)
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
