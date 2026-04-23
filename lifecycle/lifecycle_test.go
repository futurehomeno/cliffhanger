package lifecycle_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestGetAppState_DefaultFields(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)
	states := l.AllStates()

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

func TestGetState_ByType(t *testing.T) {
	t.Parallel()

	l := lifecycle.New(nil)
	l.SetAppState(lifecycle.AppStateRunning, nil)
	l.SetConfigState(lifecycle.ConfigStateConfigured)
	l.SetAuthState(lifecycle.AuthStateAuthenticated)
	l.SetConnState(lifecycle.ConnStateConnected)

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

			assert.Equal(t, tc.want, l.State(tc.stateType))
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

	l.SetConnState(lifecycle.ConnStateConnected)

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

type stubRestartsStore struct {
	count int
	err   error
}

func (s *stubRestartsStore) IncrementRestartsCount() (int, error) {
	if s.err != nil {
		return 0, s.err
	}

	s.count++

	return s.count, nil
}

func TestNew_WithStore_SetsRestartsCount(t *testing.T) {
	t.Parallel()

	store := &stubRestartsStore{count: 4}

	l := lifecycle.New(store)

	assert.Equal(t, 5, l.RestartsCount())
}

func TestNew_WithStore_StoreError_RestartsCountIsZero(t *testing.T) {
	t.Parallel()

	store := &stubRestartsStore{err: errors.New("boom")}

	l := lifecycle.New(store)

	assert.Equal(t, 0, l.RestartsCount())
}
