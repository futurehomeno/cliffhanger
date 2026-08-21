package lifecycle

import (
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/event"
)

const (
	StateTypeAppHealth   StateType = "APP_HEALTH"
	StateTypeConfigState StateType = "CONFIG_STATE"
	StateTypeAuthState   StateType = "AUTH_STATE"
	StateTypeConnState   StateType = "CONN_STATE"

	AppHealthStarting      State = "STARTING"
	AppHealthStartupError  State = "STARTUP_ERROR"
	AppHealthNotConfigured State = "NOT_CONFIGURED"
	AppHealthError         State = "ERROR"
	AppHealthRunning       State = "RUNNING"
	AppHealthTerminate     State = "TERMINATING"

	ConfigStateNotConfigured  State = "NOT_CONFIGURED"
	ConfigStateConfigured     State = "CONFIGURED"
	ConfigStatePartConfigured State = "PART_CONFIGURED"
	ConfigStateInProgress     State = "IN_PROGRESS"
	ConfigStateNA             State = "NA"

	AuthStateNotAuthenticated State = "NOT_AUTHENTICATED"
	AuthStateAuthenticated    State = "AUTHENTICATED"
	AuthStateInProgress       State = "IN_PROGRESS"
	AuthStateError            State = "ERROR"
	AuthStateLost             State = "LOST"
	AuthStateNA               State = "NA"

	ConnStateConnecting   State = "CONNECTING"
	ConnStateConnected    State = "CONNECTED"
	ConnStateDisconnected State = "DISCONNECTED"
	ConnStateNA           State = "NA"
)

type StateType string

type State string

type AppStateT struct {
	Health     string `json:"app"`
	Connection string `json:"connection"`
	Config     string `json:"config"`
	Auth       string `json:"auth"`
}

type SystemEvent struct {
	Type   StateType
	State  State
	Params map[string]string
}

type SystemEventChannel chan SystemEvent

type Lifecycle struct {
	lock            *sync.RWMutex
	systemEventBus  *event.Bus[SystemEvent]
	appHealth       State
	connectionState State
	authState       State
	configState     State
	startTime       time.Time
	restartsCount   int
}

func New(store *config.DefaultStore) *Lifecycle {
	l := &Lifecycle{
		systemEventBus:  event.NewBus[SystemEvent](),
		lock:            &sync.RWMutex{},
		appHealth:       AppHealthStarting,
		authState:       AuthStateNA,
		configState:     ConfigStateNotConfigured,
		connectionState: ConnStateNA,
		startTime:       time.Now(),
	}

	if store != nil {
		var err error
		l.restartsCount, err = store.IncrementRestartsCount()
		if err != nil {
			log.Errorf("Increment restart count err: %v", err)
		}
	}

	return l
}

func (l *Lifecycle) Uptime() int {
	l.lock.RLock()
	start := l.startTime
	l.lock.RUnlock()

	return int(time.Since(start).Seconds())
}

func (l *Lifecycle) RestartsCount() int {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.restartsCount
}

func (l *Lifecycle) AllStates() *AppStateT {
	l.lock.RLock()
	states := &AppStateT{
		Health:     string(l.appHealth),
		Connection: string(l.connectionState),
		Config:     string(l.configState),
		Auth:       string(l.authState),
	}
	l.lock.RUnlock()
	return states
}

func (l *Lifecycle) State(stateType StateType) State {
	l.lock.RLock()
	defer l.lock.RUnlock()

	switch stateType {
	case StateTypeAppHealth:
		return l.appHealth
	case StateTypeConfigState:
		return l.configState
	case StateTypeAuthState:
		return l.authState
	case StateTypeConnState:
		return l.connectionState
	}

	return ""
}

func (l *Lifecycle) ConfigState() State {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.configState
}

func (l *Lifecycle) SetConfigState(configState State) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if configState == l.configState {
		return
	}

	l.configState = configState

	l.emitStateChangeEvent(StateTypeConfigState, configState, nil)
}

func (l *Lifecycle) AuthState() State {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.authState
}

func (l *Lifecycle) SetAuthState(authState State) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if authState == l.authState {
		return
	}

	l.authState = authState

	l.emitStateChangeEvent(StateTypeAuthState, authState, nil)
}

func (l *Lifecycle) ConnectionState() State {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.connectionState
}

func (l *Lifecycle) SetConnState(connectionState State) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if connectionState == l.connectionState {
		return
	}

	l.connectionState = connectionState

	l.emitStateChangeEvent(StateTypeConnState, connectionState, nil)
}

// SetConnAndAuthState sets the connection and auth states atomically and emits a single
// auth-state event. Subscribers reacting to the auth event (e.g. the auth-loss watcher)
// then read a consistent bundle, and the trigger cannot be evicted from their buffer by a
// separate connection event. No connection event is emitted, so a subscriber tracking the
// connection must re-read State on any event instead of filtering on its type.
func (l *Lifecycle) SetConnAndAuthState(connectionState, authState State) {
	l.setConnAndAuthState(connectionState, authState, nil)
}

// SetConnAndAuthStateReason is SetConnAndAuthState that attaches a reason to the
// emitted auth event so the auth-loss watcher can report why the session was lost.
func (l *Lifecycle) SetConnAndAuthStateReason(connectionState, authState State, reason string) {
	l.setConnAndAuthState(connectionState, authState, map[string]string{"reason": reason})
}

func (l *Lifecycle) setConnAndAuthState(connectionState, authState State, params map[string]string) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if connectionState == l.connectionState && authState == l.authState {
		return
	}

	l.connectionState = connectionState
	l.authState = authState

	l.emitStateChangeEvent(StateTypeAuthState, authState, params)
}

// SetAppState sets app health, config, connection and auth state atomically and emits a single
// auth-state event, matching SetConnAndAuthState: a subscriber reacting to auth transitions
// (e.g. the auth-loss watcher, which filters on StateTypeAuthState) sees one consistent bundle
// instead of up to four separate emits, any of which past the first could be dropped from its
// buffer if it isn't draining fast enough. Health, config and connection produce no event of
// their own, so a subscriber tracking them must re-read State on any event, as WaitFor does.
func (l *Lifecycle) SetAppState(appHealth, configState, connectionState, authState State) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if appHealth == l.appHealth && configState == l.configState &&
		connectionState == l.connectionState && authState == l.authState {
		return
	}

	l.appHealth = appHealth
	l.configState = configState
	l.connectionState = connectionState
	l.authState = authState

	l.emitStateChangeEvent(StateTypeAuthState, authState, nil)
}

func (l *Lifecycle) AppHealth() State {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.appHealth
}

func (l *Lifecycle) SetAppHealth(appState State, params map[string]string) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if appState == l.appHealth {
		return
	}

	l.appHealth = appState

	l.emitStateChangeEvent(StateTypeAppHealth, appState, params)
}

// Subscribe registers a new subscription under the given ID and returns its channel. Unsubscribe
// closes every channel registered under that ID, so an ID is only safe to share between
// subscribers with the same lifetime.
func (l *Lifecycle) Subscribe(subID string, bufSize int) SystemEventChannel {
	return l.systemEventBus.Subscribe(subID, bufSize)
}

func (l *Lifecycle) Unsubscribe(subID string) {
	l.systemEventBus.Unsubscribe(subID)
}

// WaitFor blocks until the given state type reaches the target state.
func (l *Lifecycle) WaitFor(subID string, stateType StateType, targetState State) {
	// Subscribed before the state is read: the other order misses a transition landing between the
	// two and then blocks forever waiting for an event that was already delivered to nobody. The ID
	// is made unique so that concurrent waiters cannot close each other's channel.
	subID = subID + "-" + uuid.New().String()

	ch := l.Subscribe(subID, 5)
	defer l.Unsubscribe(subID)

	if l.State(stateType) == targetState {
		return
	}

	// Re-read rather than match the event: a bundled setter changes several states but emits one
	// event, so the awaited state can be reached by an event of another type.
	for range ch {
		if l.State(stateType) == targetState {
			return
		}
	}
}

func (l *Lifecycle) emitStateChangeEvent(stateType StateType, currentState State, params map[string]string) {
	l.systemEventBus.Publish(SystemEvent{Type: stateType, State: currentState, Params: params}, func(subID string) {
		log.Warnf("[cliff] State event channel=%s busy drop event %s/%s", subID, stateType, currentState)
	})
}
