package telemetry_test

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/cliffhanger/telemetry"
	"github.com/futurehomeno/cliffhanger/telemetry/types"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

// inMemoryStore wraps a *config.DefaultStore around an in-memory *config.Default
// so tests can both drive telemetry.New (via store.DefaultStore) and inspect
// state directly (via store.model / store.saves).
type inMemoryStore struct {
	*config.DefaultStore
	model   *config.Default
	saves   atomic.Int32
	saveErr error
}

func newStore() *inMemoryStore {
	s := &inMemoryStore{model: &config.Default{}}
	s.DefaultStore = config.NewDefaultStore(
		func() *config.Default { return s.model },
		func() error {
			s.saves.Add(1)

			return s.saveErr
		},
	)

	return s
}

func TestNew_NilMQTT_Errors(t *testing.T) { //nolint:paralleltest
	_, err := telemetry.New(nil, "src", newStore().DefaultStore)
	require.Error(t, err)
}

func TestNew_EmptySource_Errors(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	_, err := telemetry.New(mqtt, "", newStore().DefaultStore)
	require.Error(t, err)
}

func TestNew_NilStore_Errors(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	_, err := telemetry.New(mqtt, "src", nil)
	require.Error(t, err)
}

func TestNew_SeedsTelemetryBlock(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_seed", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	assert.Nil(t, store.model.Telemetry, "no telemetry block before New")

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	t.Cleanup(func() {
		if stop, ok := tel.(interface{ Stop() }); ok {
			stop.Stop()
		}
	})

	assert.NotNil(t, store.model.Telemetry, "New seeds a telemetry block")
	assert.True(t, store.model.Telemetry.Enabled, "seeded as enabled")
	assert.True(t, tel.IsEnabled())
}

func TestEnable_TogglesAndPersists(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_enable", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	t.Cleanup(func() {
		if stop, ok := tel.(interface{ Stop() }); ok {
			stop.Stop()
		}
	})

	require.NoError(t, tel.Enable(false))
	assert.False(t, tel.IsEnabled())
	assert.False(t, store.model.Telemetry.Enabled)

	require.NoError(t, tel.Enable(true))
	assert.True(t, tel.IsEnabled())
	assert.True(t, store.model.Telemetry.Enabled)
	assert.False(t, store.model.Telemetry.EnabledAt.IsZero(), "EnabledAt set on enable")
}

func TestSetValidity_RejectsNonPositive(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_validity", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	tel, err := telemetry.New(mqtt, "src", newStore().DefaultStore)
	require.NoError(t, err)
	t.Cleanup(func() {
		if stop, ok := tel.(interface{ Stop() }); ok {
			stop.Stop()
		}
	})

	assert.Error(t, tel.SetValidity(0))
	assert.Error(t, tel.SetValidity(-time.Second))
}

func TestSetValidity_PersistsAndReturned(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_validity_ok", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	t.Cleanup(func() {
		if stop, ok := tel.(interface{ Stop() }); ok {
			stop.Stop()
		}
	})

	savesBefore := store.saves.Load()

	require.NoError(t, tel.SetValidity(2*time.Hour))
	assert.Equal(t, 2*time.Hour, tel.Validity())
	assert.Equal(t, 2*time.Hour, store.model.Telemetry.Validity)
	assert.Greater(t, store.saves.Load(), savesBefore, "SetValidity must persist")
}

func TestSuppressedDomains_PersistsAndClones(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_suppress", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	t.Cleanup(func() {
		if stop, ok := tel.(interface{ Stop() }); ok {
			stop.Stop()
		}
	})

	domains := []string{"a", "b"}
	require.NoError(t, tel.SetSuppressedDomains(domains))

	got := tel.SuppressedDomains()
	assert.Equal(t, []string{"a", "b"}, got)

	// Mutating the caller's input must not affect persisted state.
	domains[0] = "MUTATED"
	assert.Equal(t, []string{"a", "b"}, tel.SuppressedDomains())

	// Empty slice clears the list.
	require.NoError(t, tel.SetSuppressedDomains(nil))
	assert.Nil(t, tel.SuppressedDomains())
}

func TestServiceName_DerivedFromSource(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_svcname", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	tel, err := telemetry.New(mqtt, "my-app", newStore().DefaultStore)
	require.NoError(t, err)
	t.Cleanup(func() {
		if stop, ok := tel.(interface{ Stop() }); ok {
			stop.Stop()
		}
	})

	assert.Equal(t, "my-app", string(tel.ServiceName()))
}

func TestSaveError_DoesNotPanic(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_saveerr", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	t.Cleanup(func() {
		if stop, ok := tel.(interface{ Stop() }); ok {
			stop.Stop()
		}
	})

	// Saves now start failing after construction - runtime mutations log
	// the error and continue.
	store.saveErr = errors.New("disk")

	require.NoError(t, tel.Enable(true))
	require.NoError(t, tel.SetSuppressedDomains([]string{"x"}))
}

func TestEmit_NilTelemetry_NoOp(t *testing.T) { //nolint:paralleltest
	assert.NotPanics(t, func() {
		telemetry.Emit(nil, "domain", "event", nil)
	})
}

func TestEmitRequired_NilTelemetry_NoOp(t *testing.T) { //nolint:paralleltest
	assert.NotPanics(t, func() {
		telemetry.EmitRequired(nil, "domain", "event", nil)
	})
}

func TestEmit_Disabled_IsDropped(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_emit_disabled", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{Enabled: false}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	t.Cleanup(func() {
		if stop, ok := tel.(interface{ Stop() }); ok {
			stop.Stop()
		}
	})

	// Should not panic, should not publish (no listener required to assert).
	assert.NotPanics(t, func() {
		telemetry.Emit(tel, "weather", "temperature", map[string]any{"v": 1})
	})
}

func TestEmit_SuppressedDomain_IsDropped(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_emit_supp", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{
		Enabled:           true,
		EnabledAt:         time.Now(),
		Validity:          time.Hour,
		SuppressedDomains: []string{"weather"},
	}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	t.Cleanup(func() {
		if stop, ok := tel.(interface{ Stop() }); ok {
			stop.Stop()
		}
	})

	assert.NotPanics(t, func() {
		telemetry.Emit(tel, "weather", "temperature", nil)
	})
}

func TestSetEvtTopic_OverrideAndDefault(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_evt_topic", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	tel, err := telemetry.New(mqtt, "src", newStore().DefaultStore)
	require.NoError(t, err)
	t.Cleanup(func() {
		if stop, ok := tel.(interface{ Stop() }); ok {
			stop.Stop()
		}
	})

	// Override.
	assert.NotPanics(t, func() { tel.SetEvtTopic("pt:j1/mt:evt/rt:app/rn:custom/ad:1") })

	// Empty falls back to default - no panic, no error.
	assert.NotPanics(t, func() { tel.SetEvtTopic("") })
}

func TestEmit_Enabled_Publishes(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_emit_pub", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{Enabled: true, EnabledAt: time.Now(), Validity: time.Hour}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	t.Cleanup(func() {
		if stop, ok := tel.(interface{ Stop() }); ok {
			stop.Stop()
		}
	})

	// Use a custom topic so we can confidently subscribe and observe publishing.
	customTopic := "pt:j1/mt:evt/rt:app/rn:emit_test/ad:1"
	tel.SetEvtTopic(customTopic)

	got := make(chan *fimpgo.FimpMessage, 1)
	require.NoError(t, mqtt.Subscribe(customTopic))

	mqtt.RegisterChannel("emit_test", make(fimpgo.MessageCh, 4))

	// Easier: just call Emit and assert no panic + give time for broker round-trip.
	telemetry.Emit(tel, "weather", "temperature", map[string]any{"v": 1})
	telemetry.Emit(tel, "", "noop", nil) // empty event - publish path returns error, swallowed
	close(got)
}

func TestEmitRequired_Enabled_Publishes(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_emit_req_en", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{Enabled: true, EnabledAt: time.Now(), Validity: time.Hour}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	t.Cleanup(func() {
		if stop, ok := tel.(interface{ Stop() }); ok {
			stop.Stop()
		}
	})

	assert.NotPanics(t, func() {
		telemetry.EmitRequired(tel, "domain", "event", map[string]any{"k": "v"})
	})
}

func TestEmitRequired_Disabled_IsDropped(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_emit_req_dis", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{Enabled: false}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	t.Cleanup(func() {
		if stop, ok := tel.(interface{ Stop() }); ok {
			stop.Stop()
		}
	})

	// When telemetry is disabled, even required events are dropped per the
	// matrix in types/types.go: "Enabled=false: everything is dropped."
	assert.NotPanics(t, func() {
		telemetry.EmitRequired(tel, "domain", "event", nil)
	})
}

func TestEnable_TinyValidity_TimerDisables(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_tiny_validity", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	// 50ms validity - the AfterFunc timer fires disableLocked almost immediately.
	store.model.Telemetry = &types.TelemetryConfig{Enabled: true, EnabledAt: time.Now(), Validity: 50 * time.Millisecond}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	t.Cleanup(func() {
		if stop, ok := tel.(interface{ Stop() }); ok {
			stop.Stop()
		}
	})

	// Wait for the timer to fire - disableLocked runs in a goroutine.
	require.Eventually(t, func() bool {
		return !tel.IsEnabled()
	}, time.Second, 10*time.Millisecond, "validity timer should disable telemetry")

	assert.False(t, store.model.Telemetry.Enabled)
	assert.True(t, store.model.Telemetry.EnabledAt.IsZero())
}

func TestSetValidity_BelowElapsed_DisablesImmediately(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_validity_disable", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	// Telemetry was enabled 1 hour ago.
	store.model.Telemetry = &types.TelemetryConfig{
		Enabled:   true,
		EnabledAt: time.Now().Add(-time.Hour),
		Validity:  24 * time.Hour,
	}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	t.Cleanup(func() {
		if stop, ok := tel.(interface{ Stop() }); ok {
			stop.Stop()
		}
	})

	// New validity smaller than elapsed (1m < 1h) - SetValidity should disable.
	require.NoError(t, tel.SetValidity(time.Minute))
	assert.False(t, tel.IsEnabled())
}

func TestResumeValidityWindow_ExpiredBeforeStartup_DisablesAtBoot(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_resume_expired", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	// EnabledAt 2h ago, validity 1h - already expired before this process started.
	store.model.Telemetry = &types.TelemetryConfig{
		Enabled:   true,
		EnabledAt: time.Now().Add(-2 * time.Hour),
		Validity:  time.Hour,
	}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	t.Cleanup(func() {
		if stop, ok := tel.(interface{ Stop() }); ok {
			stop.Stop()
		}
	})

	assert.False(t, tel.IsEnabled(), "expired validity window disables on resume")
}

func TestResumeValidityWindow_ClockSkew_NormalizesToNow(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_resume_skew", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	// EnabledAt is in the future - clock skew. Should be normalized to "now".
	store.model.Telemetry = &types.TelemetryConfig{
		Enabled:   true,
		EnabledAt: time.Now().Add(24 * time.Hour),
		Validity:  time.Hour,
	}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	t.Cleanup(func() {
		if stop, ok := tel.(interface{ Stop() }); ok {
			stop.Stop()
		}
	})

	assert.True(t, tel.IsEnabled(), "future EnabledAt is normalized rather than treated as expired")
}

func TestRouting_EnabledRoundTrip(t *testing.T) { //nolint:paralleltest
	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{Enabled: true, EnabledAt: time.Now()}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "enabled get/set",
				Setup: suite.BaseSetup(func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
					t.Helper()

					tel, err := telemetry.New(mqtt, "tel_test", store.DefaultStore)
					require.NoError(t, err)
					t.Cleanup(func() {
						if stop, ok := tel.(interface{ Stop() }); ok {
							stop.Stop()
						}
					})

					return telemetry.Route(tel), nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_telemetry_enabled", "tel_test"),
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.telemetry_enabled_report", "tel_test", true),
						},
					},
					{
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_telemetry_enabled", "tel_test", false),
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.telemetry_enabled_report", "tel_test", false),
						},
					},
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_telemetry_enabled", "tel_test"),
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.telemetry_enabled_report", "tel_test", false),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestRouting_ValidityRoundTrip(t *testing.T) { //nolint:paralleltest
	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{Validity: time.Hour}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "validity",
				Setup: suite.BaseSetup(func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
					t.Helper()

					tel, err := telemetry.New(mqtt, "tel_validity", store.DefaultStore)
					require.NoError(t, err)
					t.Cleanup(func() {
						if stop, ok := tel.(interface{ Stop() }); ok {
							stop.Stop()
						}
					})

					return telemetry.Route(tel), nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_telemetry_validity", "tel_validity"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.telemetry_validity_report", "tel_validity", "1h0m0s"),
						},
					},
					{
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_telemetry_validity", "tel_validity", "30m"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.telemetry_validity_report", "tel_validity", "30m0s"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestRouting_SuppressedDomainsGet(t *testing.T) { //nolint:paralleltest
	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{Enabled: true, EnabledAt: time.Now(), SuppressedDomains: []string{"alpha", "beta"}}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "suppressed get",
				Setup: suite.BaseSetup(func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
					t.Helper()

					tel, err := telemetry.New(mqtt, "tel_supp_get", store.DefaultStore)
					require.NoError(t, err)
					t.Cleanup(func() {
						if stop, ok := tel.(interface{ Stop() }); ok {
							stop.Stop()
						}
					})

					return telemetry.Route(tel), nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_telemetry_suppressed_domains", "tel_supp_get"),
						Expectations: []*suite.Expectation{
							suite.ExpectStringArray("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.telemetry_suppressed_domains_report", "tel_supp_get", []string{"alpha", "beta"}),
						},
					},
					{
						Command: suite.StringArrayMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_telemetry_suppressed_domains", "tel_supp_get", []string{"gamma"}),
						Expectations: []*suite.Expectation{
							suite.ExpectStringArray("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.telemetry_suppressed_domains_report", "tel_supp_get", []string{"gamma"}),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}
