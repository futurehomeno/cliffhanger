package lifecycle

import (
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/lifecycle/restartsstore"
	log "github.com/sirupsen/logrus"
)

const (
	StateTypeAppState    StateType = "APP_STATE"
	StateTypeConfigState StateType = "CONFIG_STATE"
	StateTypeAuthState   StateType = "AUTH_STATE"
	StateTypeConnState   StateType = "CONN_STATE"

	AppStateStarting      State = "STARTING"
	AppStateStartupError  State = "STARTUP_ERROR"
	AppStateNotConfigured State = "NOT_CONFIGURED"
	AppStateError         State = "ERROR"
	AppStateRunning       State = "RUNNING"
	AppStateTerminate     State = "TERMINATING"

	ConfigStateNotConfigured  State = "NOT_CONFIGURED"
	ConfigStateConfigured     State = "CONFIGURED"
	ConfigStatePartConfigured State = "PART_CONFIGURED"
	ConfigStateInProgress     State = "IN_PROGRESS"
	ConfigStateNA             State = "NA"

	AuthStateNotAuthenticated State = "NOT_AUTHENTICATED"
	AuthStateAuthenticated    State = "AUTHENTICATED"
	AuthStateInProgress       State = "IN_PROGRESS"
	AuthStateNA               State = "NA"

	ConnStateConnecting   State = "CONNECTING"
	ConnStateConnected    State = "CONNECTED"
	ConnStateDisconnected State = "DISCONNECTED"
	ConnStateNA           State = "NA"
)

type StateType string

type State string

type AppStates struct {
	App        string `json:"app"`
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
	lock               *sync.RWMutex
	systemEventBusLock *sync.RWMutex
	systemEventBus     map[string]SystemEventChannel
	appState           State
	connectionState    State
	authState          State
	configState        State
	startTime          time.Time
	restartsCount      int
}

func New(store restartsstore.RestartsStore) *Lifecycle {
	l := &Lifecycle{
		systemEventBus:     make(map[string]SystemEventChannel),
		lock:               &sync.RWMutex{},
		systemEventBusLock: &sync.RWMutex{},
		appState:           AppStateStarting,
		authState:          AuthStateNA,
		configState:        ConfigStateNotConfigured,
		connectionState:    ConnStateNA,
		startTime:          time.Now(),
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

func (l *Lifecycle) AllStates() *AppStates {
	l.lock.RLock()
	states := &AppStates{
		App:        string(l.appState),
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
	case StateTypeAppState:
		return l.appState
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

func (l *Lifecycle) SetConnectionState(connectionState State) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if connectionState == l.connectionState {
		return
	}

	l.connectionState = connectionState

	l.emitStateChangeEvent(StateTypeConnState, connectionState, nil)
}

func (l *Lifecycle) AppState() State {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.appState
}

func (l *Lifecycle) SetAppState(appState State, params map[string]string) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if appState == l.appState {
		return
	}

	l.appState = appState

	l.emitStateChangeEvent(StateTypeAppState, appState, params)
}

func (l *Lifecycle) Subscribe(subID string, bufSize int) SystemEventChannel {
	l.systemEventBusLock.Lock()
	defer l.systemEventBusLock.Unlock()

	// Returning already existing subscription channel if it exists.
	if _, ok := l.systemEventBus[subID]; ok {
		return l.systemEventBus[subID]
	}

	msgChan := make(SystemEventChannel, bufSize)

	l.systemEventBus[subID] = msgChan

	return msgChan
}

func (l *Lifecycle) Unsubscribe(subID string) {
	l.systemEventBusLock.Lock()
	defer l.systemEventBusLock.Unlock()

	if _, ok := l.systemEventBus[subID]; !ok {
		return
	}

	delete(l.systemEventBus, subID)
}

func (l *Lifecycle) WaitFor(subID string, stateType StateType, targetState State) {
	if l.State(stateType) == targetState {
		return
	}

	ch := l.Subscribe(subID, 5)

	for event := range ch {
		if event.Type == stateType && event.State == targetState {
			l.Unsubscribe(subID)
			return
		}
	}
}

func (l *Lifecycle) emitStateChangeEvent(stateType StateType, currentState State, params map[string]string) {
	l.systemEventBusLock.RLock()
	defer l.systemEventBusLock.RUnlock()

	for i, ch := range l.systemEventBus {
		select {
		case ch <- SystemEvent{Type: stateType, State: currentState, Params: params}:
		default:
			log.Warnf("[cliff] State event channel=%s busy drop event %s/%s", i, stateType, currentState)
		}
	}
}
