package lifecycle

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

// Constants defining event types and application states.
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

// StateType is a type representing a type of an application state.
type StateType string

// State represents one of the application states.
type State string

// AppStates is an object representing current application states.
type AppStates struct {
	App           string `json:"app"`
	Connection    string `json:"connection"`
	Config        string `json:"config"`
	Auth          string `json:"auth"`
	LastErrorText string `json:"last_error_text"`
	LastErrorCode string `json:"last_error_code"`
}

// SystemEvent is an object representing a particular system event.
type SystemEvent struct {
	Type   StateType
	State  State
	Params map[string]string
}

// SystemEventChannel is a channel used to subscribe to system events.
type SystemEventChannel chan SystemEvent

// Lifecycle is a service holding central information concerning the state of the edge application.
type Lifecycle struct {
	lock            *sync.RWMutex
	chLock          *sync.RWMutex
	systemEventBus  map[string]SystemEventChannel
	appState        State
	connectionState State
	authState       State
	configState     State
}

// New creates new instance of a lifecycle service.
func New() *Lifecycle {
	return &Lifecycle{
		systemEventBus:  make(map[string]SystemEventChannel),
		lock:            &sync.RWMutex{},
		chLock:          &sync.RWMutex{},
		appState:        AppStateStarting,
		authState:       AuthStateNA,
		configState:     ConfigStateNotConfigured,
		connectionState: ConnStateNA,
	}
}

// GetAllStates returns all application states.
func (l *Lifecycle) GetAllStates() *AppStates {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return &AppStates{
		App:           string(l.appState),
		Connection:    string(l.connectionState),
		Config:        string(l.configState),
		Auth:          string(l.authState),
		LastErrorText: "",
		LastErrorCode: "",
	}
}

// GetState returns a current application state of the provided type.
func (l *Lifecycle) GetState(stateType StateType) State {
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

// ConfigState returns current configuration state.
func (l *Lifecycle) ConfigState() State {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.configState
}

// SetConfigState sets configuration state.
func (l *Lifecycle) SetConfigState(configState State) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if configState == l.configState {
		return
	}

	l.configState = configState

	l.emitStateChangeEvent(StateTypeConfigState, configState, nil)
}

// AuthState returns current authorization state.
func (l *Lifecycle) AuthState() State {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.authState
}

// SetAuthState sets authorization state.
func (l *Lifecycle) SetAuthState(authState State) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if authState == l.authState {
		return
	}

	l.authState = authState

	l.emitStateChangeEvent(StateTypeAuthState, authState, nil)
}

// ConnectionState returns current connection state.
func (l *Lifecycle) ConnectionState() State {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.connectionState
}

// SetConnectionState sets connection state.
func (l *Lifecycle) SetConnectionState(connectionState State) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if connectionState == l.connectionState {
		return
	}

	l.connectionState = connectionState

	l.emitStateChangeEvent(StateTypeConnState, connectionState, nil)
}

// AppState returns current application state.
func (l *Lifecycle) AppState() State {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.appState
}

// SetAppState sets application state.
func (l *Lifecycle) SetAppState(appState State, params map[string]string) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if appState == l.appState {
		return
	}

	l.appState = appState

	l.emitStateChangeEvent(StateTypeAppState, appState, params)
}

// Subscribe subscribes to system events. If subscription already exists previously set channel is being returned.
func (l *Lifecycle) Subscribe(subID string, bufSize int) SystemEventChannel {
	l.chLock.Lock()
	defer l.chLock.Unlock()

	// Returning already existing subscription channel if it exists.
	if _, ok := l.systemEventBus[subID]; ok {
		return l.systemEventBus[subID]
	}

	msgChan := make(SystemEventChannel, bufSize)

	l.systemEventBus[subID] = msgChan

	return msgChan
}

// Unsubscribe removes subscription to system events.
func (l *Lifecycle) Unsubscribe(subID string) {
	l.chLock.Lock()
	defer l.chLock.Unlock()

	if _, ok := l.systemEventBus[subID]; !ok {
		return
	}

	delete(l.systemEventBus, subID)
}

// WaitFor blocks until application lifecycle state is reached.
func (l *Lifecycle) WaitFor(subID string, stateType StateType, targetState State) {
	if l.GetState(stateType) == targetState {
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

// emitStateChangeEvent emits a state change event.
func (l *Lifecycle) emitStateChangeEvent(stateType StateType, currentState State, params map[string]string) {
	l.chLock.RLock()
	defer l.chLock.RUnlock()

	for i, ch := range l.systemEventBus {
		select {
		case ch <- SystemEvent{Type: stateType, State: currentState, Params: params}:
		default:
			log.Warnf("[cliff] State event listener for ID=%s busy, event state change %s->%s dropped", i, currentState, stateType)
		}
	}
}
