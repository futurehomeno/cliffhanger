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
	cliffstorage "github.com/futurehomeno/cliffhanger/storage"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/cliffhanger/telemetry"
	"github.com/futurehomeno/cliffhanger/telemetry/types"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

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

func stopTel(t *testing.T, tel telemetry.Telemetry) {
	t.Helper()
	t.Cleanup(func() {
		if stop, ok := tel.(interface{ Stop() }); ok {
			stop.Stop()
		}
	})
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
	stopTel(t, tel)

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
	stopTel(t, tel)

	require.NoError(t, tel.Enable(false))
	assert.False(t, tel.IsEnabled())
	assert.False(t, store.model.Telemetry.Enabled)

	require.NoError(t, tel.Enable(true))
	assert.True(t, tel.IsEnabled())
	assert.True(t, store.model.Telemetry.Enabled)
	assert.False(t, store.model.Telemetry.EnabledAt.IsZero(), "EnabledAt set on enable")
}

func TestEnable_RepeatedTrue_ExtendsValidityWindow(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_enable_extend", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

	require.NoError(t, tel.Enable(true))
	first := store.model.Telemetry.EnabledAt
	require.False(t, first.IsZero())

	time.Sleep(20 * time.Millisecond)

	require.NoError(t, tel.Enable(true))
	second := store.model.Telemetry.EnabledAt

	assert.True(t, second.After(first), "Enable(true) must refresh EnabledAt on each call (heartbeat)")
	assert.True(t, tel.IsEnabled())
}

func TestSetValidity_RejectsNonPositive(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_validity", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	tel, err := telemetry.New(mqtt, "src", newStore().DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

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
	stopTel(t, tel)

	savesBefore := store.saves.Load()

	require.NoError(t, tel.SetValidity(2*time.Hour))
	assert.Equal(t, 2*time.Hour, tel.Validity())
	assert.Equal(t, 2*time.Hour, store.model.Telemetry.Validity)
	assert.Greater(t, store.saves.Load(), savesBefore, "SetValidity must persist")
}

func TestSuppressed_PersistsAndClones(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_suppress", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

	// Set suppressed entry for this app.
	suppressed := map[string]types.SuppressedEntry{
		"src": {Domains: []string{"a"}, Events: []string{"b.c"}},
	}
	require.NoError(t, tel.SetSuppressed(suppressed))

	got := tel.Suppressed()
	assert.Equal(t, []string{"a"}, got["src"].Domains)
	assert.Equal(t, []string{"b.c"}, got["src"].Events)

	// Mutating caller's input must not affect stored state.
	suppressed["src"] = types.SuppressedEntry{Domains: []string{"MUTATED"}}
	got2 := tel.Suppressed()
	assert.Equal(t, []string{"a"}, got2["src"].Domains)

	// Key absent → nil (no suppression).
	require.NoError(t, tel.SetSuppressed(map[string]types.SuppressedEntry{}))
	assert.Nil(t, store.model.Telemetry.Suppressed)

	// Entry with nil Domains+Events → suppress all.
	require.NoError(t, tel.SetSuppressed(map[string]types.SuppressedEntry{"src": {}}))
	assert.NotNil(t, store.model.Telemetry.Suppressed)
	assert.Nil(t, store.model.Telemetry.Suppressed.Domains)
	assert.Nil(t, store.model.Telemetry.Suppressed.Events)
}

func TestSuppressed_NoSuppression_ReturnsEmptyMap(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_supp_empty", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

	got := tel.Suppressed()
	assert.Empty(t, got, "Suppressed() with no suppression must return an empty map, not a zero entry that maps to 'suppress all'")

	// Round-trip must not flip 'no suppression' into 'suppress all'.
	require.NoError(t, tel.SetSuppressed(got))
	assert.Nil(t, store.model.Telemetry.Suppressed, "Set(Get()) with no suppression must remain no-suppression")
}

func TestServiceName_DerivedFromSource(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_svcname", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	tel, err := telemetry.New(mqtt, "my-app", newStore().DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

	assert.Equal(t, "my-app", string(tel.ServiceName()))
}

func TestSaveError_PropagatesToCaller(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_saveerr", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

	store.saveErr = errors.New("disk")

	assert.Error(t, tel.Enable(true), "Enable must propagate persist failure")
	assert.Error(t, tel.SetSuppressed(map[string]types.SuppressedEntry{"src": {}}), "SetSuppressed must propagate persist failure")
	assert.Error(t, tel.SetValidity(time.Hour), "SetValidity must propagate persist failure")
}

func TestEmit_NilTelemetry_NoOp(t *testing.T) { //nolint:paralleltest
	assert.NotPanics(t, func() {
		telemetry.Emit(nil, "domain", "event", nil)
	})
}

func TestEmitOnChange_NilTelemetry_NoOp(t *testing.T) { //nolint:paralleltest
	assert.NotPanics(t, func() {
		telemetry.EmitOnChange(nil, "domain", "event", nil, time.Second)
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
	stopTel(t, tel)

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
		Enabled:    true,
		EnabledAt:  time.Now(),
		Validity:   time.Hour,
		Suppressed: &types.SuppressedEntry{Domains: []string{"weather"}},
	}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

	assert.NotPanics(t, func() {
		telemetry.Emit(tel, "weather", "temperature", nil)
	})
}

func TestEmit_SuppressedEvent_IsDropped(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_emit_supp_evt", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{
		Enabled:    true,
		EnabledAt:  time.Now(),
		Validity:   time.Hour,
		Suppressed: &types.SuppressedEntry{Events: []string{"temperature"}},
	}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

	assert.NotPanics(t, func() {
		telemetry.Emit(tel, "weather", "temperature", nil)
	})
}

func TestEmit_EmptySuppressedEntry_DropsAll(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_emit_supp_all", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{Enabled: true, EnabledAt: time.Now(), Validity: time.Hour}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

	customTopic := "pt:j1/mt:evt/rt:app/rn:supp_all/ad:1"
	tel.SetEvtTopic(customTopic)
	require.NoError(t, mqtt.Subscribe(customTopic))

	ch := make(fimpgo.MessageCh, 4)
	mqtt.RegisterChannel("supp_all_ch", ch)
	t.Cleanup(func() { mqtt.UnregisterChannel("supp_all_ch") })

	drainCh := func() {
		for {
			select {
			case <-ch:
			default:
				return
			}
		}
	}

	assertAllDropped := func(label string) {
		t.Helper()
		drainCh()
		telemetry.Emit(tel, "weather", "temperature", nil)
		telemetry.Emit(tel, "energy", "consumption", nil)
		telemetry.Emit(tel, "", "standalone_event", nil)
		select {
		case msg := <-ch:
			t.Fatalf("%s: must drop all emits, got %q/%q", label, msg.Topic, msg.Payload.Interface)
		case <-time.After(300 * time.Millisecond):
		}
	}

	// Both fields nil → suppress all.
	require.NoError(t, tel.SetSuppressed(map[string]types.SuppressedEntry{"src": {}}))
	require.NotNil(t, store.model.Telemetry.Suppressed)
	assert.Nil(t, store.model.Telemetry.Suppressed.Domains)
	assert.Nil(t, store.model.Telemetry.Suppressed.Events)
	assertAllDropped("nil Domains+Events")

	// Both fields explicitly empty slices → also suppress all (len == 0 semantics).
	require.NoError(t, tel.SetSuppressed(map[string]types.SuppressedEntry{"src": {Domains: []string{}, Events: []string{}}}))
	require.NotNil(t, store.model.Telemetry.Suppressed)
	assertAllDropped("empty-slice Domains+Events")

	// Sanity check: clearing the entry lets emits through again.
	require.NoError(t, tel.SetSuppressed(map[string]types.SuppressedEntry{}))
	assert.Nil(t, store.model.Telemetry.Suppressed, "key absent must clear stored entry")
	telemetry.Emit(tel, "weather", "temperature", nil)
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("after clearing suppression, emit should publish")
	}
}

func TestEmit_NilSuppressed_EmitsNormally(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_emit_nil_supp", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{
		Enabled:   true,
		EnabledAt: time.Now(),
		Validity:  time.Hour,
		// Suppressed nil = no suppression
	}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

	customTopic := "pt:j1/mt:evt/rt:app/rn:nil_supp_test/ad:1"
	tel.SetEvtTopic(customTopic)
	require.NoError(t, mqtt.Subscribe(customTopic))

	ch := make(fimpgo.MessageCh, 4)
	mqtt.RegisterChannel("nil_supp_ch", ch)
	t.Cleanup(func() { mqtt.UnregisterChannel("nil_supp_ch") })

	telemetry.Emit(tel, "weather", "temperature", nil)

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("expected message not received — nil Suppressed should emit")
	}
}

func TestEmit_DomainSuppressedThenUnsuppressed_ResumesEmitting(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_domain_unsuppress", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{
		Enabled:    true,
		EnabledAt:  time.Now(),
		Validity:   time.Hour,
		Suppressed: &types.SuppressedEntry{Domains: []string{"weather"}},
	}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

	customTopic := "pt:j1/mt:evt/rt:app/rn:domain_unsuppress/ad:1"
	tel.SetEvtTopic(customTopic)
	require.NoError(t, mqtt.Subscribe(customTopic))

	ch := make(fimpgo.MessageCh, 4)
	mqtt.RegisterChannel("domain_unsuppress_ch", ch)
	t.Cleanup(func() { mqtt.UnregisterChannel("domain_unsuppress_ch") })

	telemetry.Emit(tel, "weather", "temperature", nil)
	select {
	case <-ch:
		t.Fatal("emit while domain suppressed should be dropped")
	case <-time.After(200 * time.Millisecond):
	}

	require.NoError(t, tel.SetSuppressed(map[string]types.SuppressedEntry{}))

	telemetry.Emit(tel, "weather", "temperature", nil)
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("after unsuppression, emit should publish")
	}
}

func TestEmit_EnableDisableEnable_TogglesPublishing(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_toggle_enabled", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{Enabled: true, EnabledAt: time.Now(), Validity: time.Hour}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

	customTopic := "pt:j1/mt:evt/rt:app/rn:toggle_enabled/ad:1"
	tel.SetEvtTopic(customTopic)
	require.NoError(t, mqtt.Subscribe(customTopic))

	ch := make(fimpgo.MessageCh, 4)
	mqtt.RegisterChannel("toggle_enabled_ch", ch)
	t.Cleanup(func() { mqtt.UnregisterChannel("toggle_enabled_ch") })

	// Enabled → emits.
	telemetry.Emit(tel, "weather", "temperature", nil)
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("enabled telemetry must publish")
	}

	// Disabled → drops.
	require.NoError(t, tel.Enable(false))
	telemetry.Emit(tel, "weather", "temperature", nil)
	select {
	case <-ch:
		t.Fatal("disabled telemetry must drop")
	case <-time.After(200 * time.Millisecond):
	}

	// Re-enabled → emits again.
	require.NoError(t, tel.Enable(true))
	telemetry.Emit(tel, "weather", "temperature", nil)
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("re-enabled telemetry must publish")
	}
}

func TestEmit_SuppressedEvent_OtherEventsInSameDomainEmit(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_event_isolated", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{
		Enabled:    true,
		EnabledAt:  time.Now(),
		Validity:   time.Hour,
		Suppressed: &types.SuppressedEntry{Events: []string{"temperature"}},
	}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

	customTopic := "pt:j1/mt:evt/rt:app/rn:event_isolated/ad:1"
	tel.SetEvtTopic(customTopic)
	require.NoError(t, mqtt.Subscribe(customTopic))

	ch := make(fimpgo.MessageCh, 4)
	mqtt.RegisterChannel("event_isolated_ch", ch)
	t.Cleanup(func() { mqtt.UnregisterChannel("event_isolated_ch") })

	// Suppressed event from the domain → dropped.
	telemetry.Emit(tel, "weather", "temperature", nil)
	select {
	case <-ch:
		t.Fatal("suppressed event must be dropped")
	case <-time.After(200 * time.Millisecond):
	}

	// Other event in the same domain → emitted.
	telemetry.Emit(tel, "weather", "humidity", nil)
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("non-suppressed event in the same domain must publish")
	}

	// Yet another event in the same domain → emitted.
	telemetry.Emit(tel, "weather", "pressure", nil)
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("non-suppressed event in the same domain must publish")
	}
}

func TestEmitOnChange_ThrottlesWithinInterval(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_eoc_throttle", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{Enabled: true, EnabledAt: time.Now(), Validity: time.Hour}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

	customTopic := "pt:j1/mt:evt/rt:app/rn:eoc_throttle/ad:1"
	tel.SetEvtTopic(customTopic)
	require.NoError(t, mqtt.Subscribe(customTopic))

	ch := make(fimpgo.MessageCh, 4)
	mqtt.RegisterChannel("eoc_throttle_ch", ch)
	t.Cleanup(func() { mqtt.UnregisterChannel("eoc_throttle_ch") })

	telemetry.EmitOnChange(tel, "d", "e", nil, time.Hour) // first: emits
	telemetry.EmitOnChange(tel, "d", "e", nil, time.Hour) // second: throttled

	// Only one message should arrive.
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("first EmitOnChange should have published")
	}

	select {
	case <-ch:
		t.Fatal("second EmitOnChange should have been throttled")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestEmitOnChange_AllowsAfterInterval(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_eoc_allow", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{Enabled: true, EnabledAt: time.Now(), Validity: time.Hour}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

	customTopic := "pt:j1/mt:evt/rt:app/rn:eoc_allow/ad:1"
	tel.SetEvtTopic(customTopic)
	require.NoError(t, mqtt.Subscribe(customTopic))

	ch := make(fimpgo.MessageCh, 4)
	mqtt.RegisterChannel("eoc_allow_ch", ch)
	t.Cleanup(func() { mqtt.UnregisterChannel("eoc_allow_ch") })

	interval := 50 * time.Millisecond

	telemetry.EmitOnChange(tel, "d", "e", nil, interval)

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("first EmitOnChange should have published")
	}

	time.Sleep(2 * interval)

	telemetry.EmitOnChange(tel, "d", "e", nil, interval)

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("second EmitOnChange after interval should have published")
	}
}

func TestSetEvtTopic_OverrideAndDefault(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_evt_topic", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	tel, err := telemetry.New(mqtt, "src", newStore().DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

	assert.NotPanics(t, func() { tel.SetEvtTopic("pt:j1/mt:evt/rt:app/rn:custom/ad:1") })
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
	stopTel(t, tel)

	customTopic := "pt:j1/mt:evt/rt:app/rn:emit_test/ad:1"
	tel.SetEvtTopic(customTopic)

	got := make(chan *fimpgo.FimpMessage, 1)
	require.NoError(t, mqtt.Subscribe(customTopic))

	mqtt.RegisterChannel("emit_test", make(fimpgo.MessageCh, 4))

	telemetry.Emit(tel, "weather", "temperature", map[string]any{"v": 1})
	telemetry.Emit(tel, "", "noop", nil)
	close(got)
}

func TestEnable_TinyValidity_TimerDisables(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_tiny_validity", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{Enabled: true, EnabledAt: time.Now(), Validity: 50 * time.Millisecond}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

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
	store.model.Telemetry = &types.TelemetryConfig{
		Enabled:   true,
		EnabledAt: time.Now().Add(-time.Hour),
		Validity:  24 * time.Hour,
	}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

	require.NoError(t, tel.SetValidity(time.Minute))
	assert.False(t, tel.IsEnabled())
}

func TestResumeValidityWindow_ExpiredBeforeStartup_DisablesAtBoot(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_resume_expired", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{
		Enabled:   true,
		EnabledAt: time.Now().Add(-2 * time.Hour),
		Validity:  time.Hour,
	}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

	assert.False(t, tel.IsEnabled(), "expired validity window disables on resume")
}

func TestResumeValidityWindow_ClockSkew_NormalizesToNow(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("cliff_test_resume_skew", "", "", "")
	require.NoError(t, mqtt.Start(2*time.Second))
	t.Cleanup(mqtt.Stop)

	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{
		Enabled:   true,
		EnabledAt: time.Now().Add(24 * time.Hour),
		Validity:  time.Hour,
	}

	tel, err := telemetry.New(mqtt, "src", store.DefaultStore)
	require.NoError(t, err)
	stopTel(t, tel)

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
					stopTel(t, tel)

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
					stopTel(t, tel)

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

func TestRouting_SuppressedGet(t *testing.T) { //nolint:paralleltest
	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{
		Enabled:    true,
		EnabledAt:  time.Now(),
		Suppressed: &types.SuppressedEntry{Domains: []string{"alpha"}, Events: []string{"beta.x"}},
	}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "suppressed get/set",
				Setup: suite.BaseSetup(func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
					t.Helper()

					tel, err := telemetry.New(mqtt, "tel_supp_get", store.DefaultStore)
					require.NoError(t, err)
					stopTel(t, tel)

					return telemetry.Route(tel), nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_telemetry_suppressed", "tel_supp_get"),
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.telemetry_suppressed_report", "tel_supp_get",
							map[string]types.SuppressedEntry{"tel_supp_get": {Domains: []string{"alpha"}, Events: []string{"beta.x"}}}),
						},
					},
					{
						Command: suite.ObjectMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_telemetry_suppressed", "tel_supp_get",
							map[string]types.SuppressedEntry{"tel_supp_get": {Domains: []string{"gamma"}, Events: []string{}}}),
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.telemetry_suppressed_report", "tel_supp_get",
								map[string]types.SuppressedEntry{"tel_supp_get": {Domains: []string{"gamma"}, Events: []string{}}}),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestRouting_SetTelemetry_PersistsToStore(t *testing.T) { //nolint:paralleltest
	store := newStore()
	store.model.Telemetry = &types.TelemetryConfig{Enabled: true, EnabledAt: time.Now(), Validity: time.Hour}

	var (
		enableSavesBefore   int32
		validitySavesBefore int32
		suppSavesBefore     int32
	)

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "set persists",
				Setup: suite.BaseSetup(func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
					t.Helper()

					tel, err := telemetry.New(mqtt, "tel_persist", store.DefaultStore)
					require.NoError(t, err)
					stopTel(t, tel)

					return telemetry.Route(tel), nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_telemetry_enabled", "tel_persist", false),
						InitCallbacks: []suite.Callback{
							func(t *testing.T) { t.Helper(); enableSavesBefore = store.saves.Load() },
						},
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.telemetry_enabled_report", "tel_persist", false),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								require.Eventually(t, func() bool {
									return store.saves.Load() > enableSavesBefore
								}, time.Second, 10*time.Millisecond, "set_telemetry_enabled must persist")
							},
						},
					},
					{
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_telemetry_validity", "tel_persist", "30m"),
						InitCallbacks: []suite.Callback{
							func(t *testing.T) { t.Helper(); validitySavesBefore = store.saves.Load() },
						},
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.telemetry_validity_report", "tel_persist", "30m0s"),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								require.Eventually(t, func() bool {
									return store.saves.Load() > validitySavesBefore
								}, time.Second, 10*time.Millisecond, "set_telemetry_validity must persist")
							},
						},
					},
					{
						Command: suite.ObjectMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_telemetry_suppressed", "tel_persist",
							map[string]types.SuppressedEntry{"tel_persist": {Domains: []string{"alpha"}}}),
						InitCallbacks: []suite.Callback{
							func(t *testing.T) { t.Helper(); suppSavesBefore = store.saves.Load() },
						},
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.telemetry_suppressed_report", "tel_persist",
								map[string]types.SuppressedEntry{"tel_persist": {Domains: []string{"alpha"}}}),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								require.Eventually(t, func() bool {
									return store.saves.Load() > suppSavesBefore
								}, time.Second, 10*time.Millisecond, "set_telemetry_suppressed must persist")
							},
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestRouting_SetTelemetry_UpdatesConfiguredAt(t *testing.T) { //nolint:paralleltest
	dir := t.TempDir()

	cfg := &config.Default{
		Telemetry: &types.TelemetryConfig{Enabled: true, EnabledAt: time.Now(), Validity: time.Hour},
	}

	realStorage := cliffstorage.New[*config.Default](cfg, dir, "config.json")

	store := config.NewDefaultStore(
		func() *config.Default { return cfg },
		realStorage.Save,
	)

	var configuredAtBefore time.Time

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "configured_at stamped on save",
				Setup: suite.BaseSetup(func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
					t.Helper()

					tel, err := telemetry.New(mqtt, "tel_cfg_at", store)
					require.NoError(t, err)
					stopTel(t, tel)

					return telemetry.Route(tel), nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_telemetry_enabled", "tel_cfg_at", false),
						InitCallbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								configuredAtBefore = parseConfiguredAt(t, cfg.ConfiguredAt)
							},
						},
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.telemetry_enabled_report", "tel_cfg_at", false),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								require.Eventually(t, func() bool {
									return parseConfiguredAt(t, cfg.ConfiguredAt).After(configuredAtBefore)
								}, time.Second, 10*time.Millisecond, "ConfiguredAt must be stamped on save")
							},
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func parseConfiguredAt(t *testing.T, raw string) time.Time {
	t.Helper()

	if raw == "" {
		return time.Time{}
	}

	parsed, err := time.Parse(time.RFC3339Nano, raw)
	require.NoError(t, err)

	return parsed
}
