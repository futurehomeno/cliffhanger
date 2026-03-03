package adapter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/futurehomeno/cliffhanger/storage"
)

// adapterStateModel is a model of the adapter state file.
type adapterStateModel struct {
	AddressIndex int                         `json:"address_index"`
	Things       map[string]*thingStateModel `json:"things"`
}

// thingStateModel is a model of a thing state record within the adapter state file.
type thingStateModel struct {
	ID                string          `json:"id"`
	Address           string          `json:"address"`
	Info              json.RawMessage `json:"info,omitempty"`
	State             json.RawMessage `json:"state,omitempty"`
	InclusionChecksum uint32          `json:"inclusion_checksum"`
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
	storageService := storage.NewState(&adapterStateModel{}, workDir, "adapter.json")

	if err := storageService.Load(); err != nil {
		return nil, fmt.Errorf("state: failed to load the initial adapter state: %w", err)
	}

	return &state{
		Storage: storageService,
	}, nil
}

// state is a private implementation of the adapter state service.
type state struct {
	storage.Storage[*adapterStateModel]
	lock sync.RWMutex
}

// acquireAddress increments address index and returns its current value.
func (s *state) acquireAddress() (string, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.Model().AddressIndex++

	if err := s.Save(); err != nil {
		return "", fmt.Errorf("state: failed to persist address index: %w", err)
	}

	return strconv.Itoa(s.Model().AddressIndex), nil
}

// all returns all persisted thing states.
func (s *state) all() []ThingState {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var thingStates []ThingState

	for _, m := range s.Model().Things {
		thingStates = append(thingStates, newThingState(s, m))
	}

	return thingStates
}

// add persists a new thing state.
func (s *state) add(model *thingStateModel) (ThingState, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.Model().Things == nil {
		s.Model().Things = make(map[string]*thingStateModel)
	}

	s.Model().Things[model.ID] = model

	if err := s.Save(); err != nil {
		return nil, fmt.Errorf("state: failed to persist state of a thing with ID %s: %w", model.ID, err)
	}

	return newThingState(s, model), nil
}

// remove deletes a thing state at a given ID.
func (s *state) remove(id string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.Model().Things, id)

	if err := s.Save(); err != nil {
		return fmt.Errorf("state: failed to remove state of a thing with ID %s: %w", id, err)
	}

	return nil
}

// byID returns a thing state for a thing with a given ID.
func (s *state) byID(id string) ThingState {
	s.lock.RLock()
	defer s.lock.RUnlock()

	ts, ok := s.Model().Things[id]
	if !ok {
		return nil
	}

	return newThingState(s, ts)
}

// byAddress returns a thing state for a thing with a given address.
func (s *state) byAddress(address string) ThingState {
	s.lock.RLock()
	defer s.lock.RUnlock()

	for _, ts := range s.Model().Things {
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
	Info(model any) error
	// State marshals thing persisted state into the provided model.
	State(model any) error
	// SetState persists new state of a thing.
	SetState(model any) error
	// GetInclusionChecksum returns the checksum of the inclusion report stored in the thing state.
	GetInclusionChecksum() uint32
	// SetInclusionChecksum persists the checksum of the inclusion report in the thing state.
	SetInclusionChecksum(checksum uint32) error
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
func (s *thingState) Info(model any) error {
	s.state.lock.RLock()
	defer s.state.lock.RUnlock()

	if len(s.model.Info) == 0 {
		return nil
	}

	err := json.Unmarshal(s.model.Info, model)
	if err != nil {
		return fmt.Errorf("thing state: failed to unmarshal info of a thing with ID %s into a provided model: %w", s.ID(), err)
	}

	return nil
}

// State marshals thing persisted state into the provided model.
func (s *thingState) State(model any) error {
	s.state.lock.RLock()
	defer s.state.lock.RUnlock()

	if len(s.model.State) == 0 {
		return nil
	}

	err := json.Unmarshal(s.model.State, model)
	if err != nil {
		return fmt.Errorf("thing state: failed to unmarshal state of a thing with ID %s into a provided model: %w", s.ID(), err)
	}

	return nil
}

// SetState persists new state of a thing.
func (s *thingState) SetState(model any) error {
	s.state.lock.Lock()
	defer s.state.lock.Unlock()

	b, err := json.Marshal(model)
	if err != nil {
		return fmt.Errorf("thing state: failed to marshal state of a thing with ID %s from a provided model: %w", s.ID(), err)
	}

	s.model.State = b

	err = s.state.Save()
	if err != nil {
		return fmt.Errorf("thing state: failed to persist state of a thing with ID %s: %w", s.ID(), err)
	}

	return nil
}

// GetInclusionChecksum returns the checksum of the inclusion report stored in the thing state.
func (s *thingState) GetInclusionChecksum() uint32 {
	s.state.lock.RLock()
	defer s.state.lock.RUnlock()

	return s.model.InclusionChecksum
}

// SetInclusionChecksum persists the checksum of the inclusion report in the thing state.
func (s *thingState) SetInclusionChecksum(checksum uint32) error {
	s.state.lock.Lock()
	defer s.state.lock.Unlock()

	s.model.InclusionChecksum = checksum

	err := s.state.Save()
	if err != nil {
		return fmt.Errorf("thing state: failed to persist inclusion checksum of a thing with ID %s: %w", s.ID(), err)
	}

	return nil
}
