package root

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/notification"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

type fakeNotifier struct {
	mu     sync.Mutex
	events []*notification.Event
}

func (f *fakeNotifier) Event(e *notification.Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, e)

	return nil
}

func (f *fakeNotifier) Message(string) error { return nil }

func (f *fakeNotifier) EventWithProps(e *notification.Event, _ map[string]string) error {
	return f.Event(e)
}

func (f *fakeNotifier) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.events)
}

func (f *fakeNotifier) first() *notification.Event {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.events) == 0 {
		return nil
	}

	return f.events[0]
}

// TestAuthLossWatcher_NotifiesOnlyOnLostAfterAuthenticated drives real lifecycle transitions
// through the watcher: a never-authenticated app dropping to NOT_AUTHENTICATED stays silent,
// while a genuine AUTHENTICATED -> LOST transition sends exactly one notification.
func TestAuthLossWatcher_NotifiesOnlyOnLostAfterAuthenticated(t *testing.T) { //nolint:paralleltest
	mqtt := suite.DefaultMQTT("root_authloss_watch", "", "", "")
	require.NoError(t, mqtt.Start(5*time.Second))
	defer mqtt.Stop()

	lc := lifecycle.New(nil)
	notifier := &fakeNotifier{}
	a := &app{
		mqtt:             mqtt,
		lifecycle:        lc,
		resourceName:     "test_app",
		authLossNotifier: notifier,
		authLossEvent:    &notification.Event{EventName: "test_status_offline"},
	}

	a.startAuthLossWatcher(nil)
	defer a.stopAuthLossWatcher()

	lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
	lc.SetAuthState(lifecycle.AuthStateAuthenticated)
	lc.SetAuthState(lifecycle.AuthStateLost)

	require.Eventually(t, func() bool { return notifier.count() == 1 }, time.Second, 10*time.Millisecond,
		"AUTHENTICATED->LOST must send exactly one notification, NA->NOT_AUTHENTICATED none")
	assert.Equal(t, "test_status_offline", notifier.first().EventName)
}

// TestReportAuthLoss_SuppressedWhenReportingDisabled confirms the reporting gate suppresses the
// notification (and the app-state report and telemetry emitted alongside it).
func TestReportAuthLoss_SuppressedWhenReportingDisabled(t *testing.T) { //nolint:paralleltest
	notifier := &fakeNotifier{}
	a := &app{
		mqtt:                  suite.DefaultMQTT("root_authloss_off", "", "", ""),
		lifecycle:             lifecycle.New(nil),
		resourceName:          "test_app",
		authLossNotifier:      notifier,
		authLossEvent:         &notification.Event{EventName: "test_status_offline"},
		authLossReportEnabled: func() bool { return false },
	}

	a.reportAuthLoss(nil, "unauthorized")

	assert.Zero(t, notifier.count(), "disabled auth-loss reporting must suppress the notification")
}

func TestNextAuthArm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		armed      bool
		state      lifecycle.State
		wantReport bool
		wantArmed  bool
	}{
		{"authenticated arms", false, lifecycle.AuthStateAuthenticated, false, true},
		{"authenticated keeps armed", true, lifecycle.AuthStateAuthenticated, false, true},
		{"lost while armed reports and disarms", true, lifecycle.AuthStateLost, true, false},
		{"lost while disarmed is ignored", false, lifecycle.AuthStateLost, false, false},
		{"not authenticated leaves arm unchanged", true, lifecycle.AuthStateNotAuthenticated, false, true},
		{"in progress leaves arm unchanged", false, lifecycle.AuthStateInProgress, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			report, armed := nextAuthArm(tt.armed, tt.state)
			assert.Equal(t, tt.wantReport, report)
			assert.Equal(t, tt.wantArmed, armed)
		})
	}
}

func TestNextAuthArm_Sequences(t *testing.T) {
	t.Parallel()

	// A LOST only reports after an AUTHENTICATED was seen, even across an intermediate state.
	armed := false
	reports := 0
	for _, s := range []lifecycle.State{
		lifecycle.AuthStateNotAuthenticated,
		lifecycle.AuthStateAuthenticated,
		lifecycle.AuthStateInProgress,
		lifecycle.AuthStateLost,
		lifecycle.AuthStateLost, // repeated LOST must not re-fire
	} {
		var report bool
		report, armed = nextAuthArm(armed, s)
		if report {
			reports++
		}
	}

	assert.Equal(t, 1, reports, "one report for the armed loss, none for the repeat")

	// Re-authentication re-arms, so a subsequent loss reports again.
	_, armed = nextAuthArm(armed, lifecycle.AuthStateAuthenticated)
	report, _ := nextAuthArm(armed, lifecycle.AuthStateLost)
	assert.True(t, report, "a loss after re-authentication reports again")
}
