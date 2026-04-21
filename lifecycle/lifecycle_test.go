package lifecycle_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/lifecycle"
)

func TestNew_DefaultStates(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)

	assert.Equal(t, lifecycle.AppStateStarting, l.AppState())
	assert.Equal(t, lifecycle.AuthStateNA, l.AuthState())
	assert.Equal(t, lifecycle.ConfigStateNotConfigured, l.ConfigState())
	assert.Equal(t, lifecycle.ConnStateNA, l.ConnectionState())
}

func TestGetAllStates_DefaultFields(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)
	states := l.GetAllStates()

	assert.Equal(t, string(lifecycle.AppStateStarting), states.App)
	assert.Equal(t, string(lifecycle.AuthStateNA), states.Auth)
	assert.Equal(t, string(lifecycle.ConfigStateNotConfigured), states.Config)
	assert.Equal(t, string(lifecycle.ConnStateNA), states.Connection)
	assert.GreaterOrEqual(t, l.Uptime(), 0)
	assert.Equal(t, 0, l.RestartsCount())
}

func TestUptime_IncreasesOverTime(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)

	time.Sleep(1100 * time.Millisecond)

	assert.GreaterOrEqual(t, l.Uptime(), 1)
}

func TestSetRestartCount(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)
	l.SetRestartCount(7)

	assert.Equal(t, 7, l.RestartsCount())
}

func TestGetState_ByType(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)
	l.SetAppState(lifecycle.AppStateRunning, nil)
	l.SetConfigState(lifecycle.ConfigStateConfigured)
	l.SetAuthState(lifecycle.AuthStateAuthenticated)
	l.SetConnectionState(lifecycle.ConnStateConnected)

	testCases := []struct {
		stateType lifecycle.StateType
		want      lifecycle.State
	}{
		{lifecycle.StateTypeAppState, lifecycle.AppStateRunning},
		{lifecycle.StateTypeConfigState, lifecycle.ConfigStateConfigured},
		{lifecycle.StateTypeAuthState, lifecycle.AuthStateAuthenticated},
		{lifecycle.StateTypeConnState, lifecycle.ConnStateConnected},
		{"UNKNOWN", ""},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(string(tc.stateType), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, l.GetState(tc.stateType))
		})
	}
}

func TestSetAppState_EmitsEvent(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)
	ch := l.Subscribe("test", 1)

	l.SetAppState(lifecycle.AppStateRunning, nil)

	require.Eventually(t, func() bool { return len(ch) == 1 }, time.Second, 10*time.Millisecond)

	event := <-ch
	assert.Equal(t, lifecycle.StateTypeAppState, event.Type)
	assert.Equal(t, lifecycle.AppStateRunning, event.State)
}

func TestSetAppState_NoDuplicateEvent(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)
	ch := l.Subscribe("test", 5)

	l.SetAppState(lifecycle.AppStateRunning, nil)
	l.SetAppState(lifecycle.AppStateRunning, nil) // same state: no second event

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, len(ch))
}

func TestSetConfigState_EmitsEvent(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)
	ch := l.Subscribe("test", 1)

	l.SetConfigState(lifecycle.ConfigStateConfigured)

	require.Eventually(t, func() bool { return len(ch) == 1 }, time.Second, 10*time.Millisecond)

	event := <-ch
	assert.Equal(t, lifecycle.StateTypeConfigState, event.Type)
	assert.Equal(t, lifecycle.ConfigStateConfigured, event.State)
}

func TestSetAuthState_EmitsEvent(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)
	ch := l.Subscribe("test", 1)

	l.SetAuthState(lifecycle.AuthStateAuthenticated)

	require.Eventually(t, func() bool { return len(ch) == 1 }, time.Second, 10*time.Millisecond)

	event := <-ch
	assert.Equal(t, lifecycle.StateTypeAuthState, event.Type)
	assert.Equal(t, lifecycle.AuthStateAuthenticated, event.State)
}

func TestSetConnectionState_EmitsEvent(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)
	ch := l.Subscribe("test", 1)

	l.SetConnectionState(lifecycle.ConnStateConnected)

	require.Eventually(t, func() bool { return len(ch) == 1 }, time.Second, 10*time.Millisecond)

	event := <-ch
	assert.Equal(t, lifecycle.StateTypeConnState, event.Type)
	assert.Equal(t, lifecycle.ConnStateConnected, event.State)
}

func TestSubscribe_ReturnsExistingChannel(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)

	ch1 := l.Subscribe("sub", 1)
	ch2 := l.Subscribe("sub", 1)

	assert.Equal(t, ch1, ch2)
}

func TestUnsubscribe_StopsEvents(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)
	l.Subscribe("sub", 5)
	l.Unsubscribe("sub")

	l.SetAppState(lifecycle.AppStateRunning, nil)

	time.Sleep(50 * time.Millisecond)

	ch := l.Subscribe("sub", 5)
	assert.Equal(t, 0, len(ch))
}

func TestWaitFor_ReturnsImmediatelyIfAlreadyInState(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)
	l.SetAppState(lifecycle.AppStateRunning, nil)

	done := make(chan struct{})

	go func() {
		l.WaitFor("test", lifecycle.StateTypeAppState, lifecycle.AppStateRunning)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("WaitFor did not return immediately for already-reached state")
	}
}

func TestWaitFor_BlocksUntilStateReached(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)

	done := make(chan struct{})

	go func() {
		l.WaitFor("test", lifecycle.StateTypeAppState, lifecycle.AppStateRunning)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	select {
	case <-done:
		t.Fatal("WaitFor returned before state was set")
	default:
	}

	l.SetAppState(lifecycle.AppStateRunning, nil)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("WaitFor did not return after state was set")
	}
}

// stubLogStats is a test double for lifecycle.LogStatsProvider.
type stubLogStats struct {
	errors   int
	warnings int
}

func (s *stubLogStats) ErrorsCount() int   { return s.errors }
func (s *stubLogStats) WarningsCount() int { return s.warnings }

// stubRestartsStore is a test double for lifecycle.RestartsStore.
type stubRestartsStore struct {
	count    int
	getErr   error
	setErr   error
	getCalls int
	setCalls int
	lastSet  int
}

func (s *stubRestartsStore) GetRestartsCount() (int, error) {
	s.getCalls++

	if s.getErr != nil {
		return 0, s.getErr
	}

	return s.count, nil
}

func (s *stubRestartsStore) SetRestartsCount(n int) error {
	s.setCalls++
	s.lastSet = n

	if s.setErr != nil {
		return s.setErr
	}

	s.count = n

	return nil
}

func TestLoadRestartsCount_IncrementsAndPersists(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)
	store := &stubRestartsStore{count: 4}

	require.NoError(t, l.LoadRestartsCount(store))

	assert.Equal(t, 5, store.lastSet)
	assert.Equal(t, 5, store.count)
	assert.Equal(t, 5, l.RestartsCount())
}

func TestLoadRestartsCount_FromZero(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)
	store := &stubRestartsStore{}

	require.NoError(t, l.LoadRestartsCount(store))

	assert.Equal(t, 1, store.lastSet)
	assert.Equal(t, 1, l.RestartsCount())
}

func TestLoadRestartsCount_GetError(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)
	getErr := errors.New("read boom")
	store := &stubRestartsStore{count: 7, getErr: getErr}

	err := l.LoadRestartsCount(store)

	require.Error(t, err)
	assert.ErrorIs(t, err, getErr)
	assert.Equal(t, 0, store.setCalls)
	assert.Equal(t, 0, l.RestartsCount())
}

func TestLoadRestartsCount_SetError(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)
	setErr := errors.New("write boom")
	store := &stubRestartsStore{count: 2, setErr: setErr}

	err := l.LoadRestartsCount(store)

	require.Error(t, err)
	assert.ErrorIs(t, err, setErr)
	assert.Equal(t, 3, store.lastSet)
	assert.Equal(t, 0, l.RestartsCount())
}

func TestNewDefaultRestartsStore_RoundTrip(t *testing.T) {
	t.Parallel()

	d := &config.Default{}
	saves := 0
	store := lifecycle.NewDefaultRestartsStore(func() *config.Default { return d }, func() error { saves++; return nil })

	// Missing value: first startup reads zero without error.
	n, err := store.GetRestartsCount()
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, 0, saves)

	// Persisted value round-trips through the embedded Default.
	require.NoError(t, store.SetRestartsCount(4))
	assert.Equal(t, 4, d.RestartsCount)
	assert.Equal(t, 1, saves)

	got, err := store.GetRestartsCount()
	require.NoError(t, err)
	assert.Equal(t, 4, got)
}

func TestNewDefaultRestartsStore_PropagatesSaveError(t *testing.T) {
	t.Parallel()

	d := &config.Default{}
	saveErr := errors.New("disk boom")
	store := lifecycle.NewDefaultRestartsStore(func() *config.Default { return d }, func() error { return saveErr })

	err := store.SetRestartsCount(9)

	require.Error(t, err)
	assert.ErrorIs(t, err, saveErr)
}

func TestLoadRestartsCount_WithDefaultStore(t *testing.T) {
	t.Parallel()

	// End-to-end: config-backed store + Lifecycle, across a simulated restart.
	d := &config.Default{}
	store := lifecycle.NewDefaultRestartsStore(func() *config.Default { return d }, func() error { return nil })

	l1 := lifecycle.New(nil)
	require.NoError(t, l1.LoadRestartsCount(store))
	assert.Equal(t, 1, l1.RestartsCount())
	assert.Equal(t, 1, d.RestartsCount)

	// Second process (same config) sees the incremented count.
	l2 := lifecycle.New(nil)
	require.NoError(t, l2.LoadRestartsCount(store))
	assert.Equal(t, 2, l2.RestartsCount())
	assert.Equal(t, 2, d.RestartsCount)
}
