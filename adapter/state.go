package adapter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/futurehomeno/cliffhanger/storage"
)

// stateModel is a model of the adapter state file.
type stateModel struct {
	AddressIndex int                         `json:"address_index"`
	Things       map[string]*thingStateModel `json:"things"`
}

// thingStateModel is a model of a thing state record within the adapter state file.
type thingStateModel struct {
	ID      string          `json:"id"`
	Address string          `json:"address"`
	Info    json.RawMessage `json:"info,omitempty"`
	State   json.RawMessage `json:"state,omitempty"`
}

// State is an interface representing a persistent state of the adapter and its things.
type State interface {
	// acquireAddress increments address index and returns its current value.
	acquireAddress() (string, error)
	// all returns all persisted thing states.
	all() []ThingState
	// add persists a new thing state.
	add(model *thingStateModel) (ThingState, error)
	// remove deletes a thing state at a given ID.
	remove(id string) error
	// byID returns a thing state for a thing with a given ID.
	byID(id string) ThingState
	// byAddress returns a thing state for a thing with a given address.
	byAddress(address string) ThingState
}

// NewState creates new instance of the adapter state.
func NewState(workDir string) (State, error) {
	model := &stateModel{}
	storageService := storage.New(model, workDir, "adapter.json")

	if err := storageService.Load(); err != nil {
		return nil, fmt.Errorf("state: failed to load the initial adapter state: %w", err)
	}

	return &state{
		model:   model,
		storage: storageService,
		lock:    &sync.RWMutex{},
	}, nil
}

// state is a private implementation of the adapter state service.
type state struct {
	storage storage.Storage
	model   *stateModel
	lock    *sync.RWMutex
}

// acquireAddress increments address index and returns its current value.
func (s *state) acquireAddress() (string, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.model.AddressIndex++

	if err := s.storage.Save(); err != nil {
		return "", fmt.Errorf("state: failed to persist address index: %w", err)
	}

	return strconv.Itoa(s.model.AddressIndex), nil
}

// all returns all persisted thing states.
func (s *state) all() []ThingState {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var thingStates []ThingState

	for _, m := range s.model.Things {
		thingStates = append(thingStates, newThingState(s, m))
	}

	return thingStates
}

// add persists a new thing state.
func (s *state) add(model *thingStateModel) (ThingState, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.model.Things[model.ID] = model

	if err := s.storage.Save(); err != nil {
		return nil, fmt.Errorf("state: failed to persist state of a thing with ID %s: %w", model.ID, err)
	}

	return newThingState(s, model), nil
}

// remove deletes a thing state at a given ID.
func (s *state) remove(id string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.model.Things, id)

	if err := s.storage.Save(); err != nil {
		return fmt.Errorf("state: failed to remove state of a thing with ID %s: %w", id, err)
	}

	return nil
}

// byID returns a thing state for a thing with a given ID.
func (s *state) byID(id string) ThingState {
	s.lock.RLock()
	defer s.lock.RUnlock()

	ts, ok := s.model.Things[id]
	if !ok {
		return nil
	}

	return newThingState(s, ts)
}

// byAddress returns a thing state for a thing with a given address.
func (s *state) byAddress(address string) ThingState {
	s.lock.RLock()
	defer s.lock.RUnlock()

	for _, ts := range s.model.Things {
		if ts.Address == address {
			return newThingState(s, ts)
		}
	}

	return nil
}

// ThingState represents a proxy service responsible for maintaining persistent state of a thing within the adapter.
type ThingState interface {
	// ID returns the ID of the thing.
	ID() string
	// Address returns the address assigned to the thing by the adapter.
	Address() string
	// Info marshals thing optional information into the provided model.
	Info(model interface{}) error
	// State marshals thing persisted state into the provided model.
	State(model interface{}) error
	// SetState persists new state of a thing.
	SetState(model interface{}) error
}

// newThingState creates new instance of a thing state proxy service.
func newThingState(s *state, m *thingStateModel) ThingState {
	return &thingState{
		state: s,
		model: m,
	}
}

// thingState is a private implementation of a thing state service.
type thingState struct {
	state *state
	model *thingStateModel
}

// ID returns the ID of the thing.
func (s *thingState) ID() string {
	s.state.lock.RLock()
	defer s.state.lock.RUnlock()

	return s.model.ID
}

// Address returns the address assigned to the thing by the adapter.
func (s *thingState) Address() string {
	s.state.lock.RLock()
	defer s.state.lock.RUnlock()

	return s.model.Address
}

// Info marshals thing optional information into the provided model.
func (s *thingState) Info(model interface{}) error {
	s.state.lock.RLock()
	defer s.state.lock.RUnlock()

	b := s.model.Info

	err := json.Unmarshal(b, model)
	if err != nil {
		return fmt.Errorf("thing state: failed to unmarshal info of a thing with ID %s into a provided model: %w", s.ID(), err)
	}

	return nil
}

// State marshals thing persisted state into the provided model.
func (s *thingState) State(model interface{}) error {
	s.state.lock.RLock()
	defer s.state.lock.RUnlock()

	b := s.model.State

	err := json.Unmarshal(b, model)
	if err != nil {
		return fmt.Errorf("thing state: failed to unmarshal state of a thing with ID %s into a provided model: %w", s.ID(), err)
	}

	return nil
}

// SetState persists new state of a thing.
func (s *thingState) SetState(model interface{}) error {
	s.state.lock.Lock()
	defer s.state.lock.Unlock()

	b, err := json.Marshal(model)
	if err != nil {
		return fmt.Errorf("thing state: failed to marshal state of a thing with ID %s from a provided model: %w", s.ID(), err)
	}

	s.model.State = b

	err = s.state.storage.Save()
	if err != nil {
		return fmt.Errorf("thing state: failed to persist state of a thing with ID %s: %w", s.ID(), err)
	}

	return nil
}
