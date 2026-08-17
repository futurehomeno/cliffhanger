package adapter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/futurehomeno/cliffhanger/storage"
)

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
	// all returns all persisted thing states.
	all() []ThingState
	// add persists a new thing state, assigning it a new address when the model carries none.
	add(model *thingStateModel) (ThingState, error)
	// remove deletes a thing state at a given ID.
	remove(id string) error
	// byID returns a thing state for a thing with a given ID.
	byID(id string) ThingState
	// modelByID returns the raw record of a thing with a given ID, or nil when there is none.
	modelByID(id string) *thingStateModel
	// byAddress returns a thing state for a thing with a given address.
	byAddress(address string) ThingState
}

func NewState(workDir string) (State, error) {
	storageService := storage.NewState(&adapterStateModel{}, workDir, "adapter.json")

	if err := storageService.Load(); err != nil {
		return nil, fmt.Errorf("state: failed to load the initial adapter state: %w", err)
	}

	return &state{
		Storage: storageService,
	}, nil
}

type state struct {
	storage.Storage[*adapterStateModel]
	lock sync.RWMutex
}

func (s *state) all() []ThingState {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var thingStates []ThingState

	for _, m := range s.Model().Things {
		thingStates = append(thingStates, newThingState(s, m))
	}

	return thingStates
}

func (s *state) add(model *thingStateModel) (ThingState, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.Model().Things == nil {
		s.Model().Things = make(map[string]*thingStateModel)
	}

	// Assigning the address here rather than through a separate acquireAddress keeps creating a
	// thing to a single state write instead of two full rewrites of adapter.json.
	index := s.Model().AddressIndex

	if model.Address == "" {
		s.Model().AddressIndex++
		model.Address = strconv.Itoa(s.Model().AddressIndex)
	}

	old, ok := s.Model().Things[model.ID]

	s.Model().Things[model.ID] = model

	if err := s.Save(); err != nil {
		// Restore on a failed write, mirroring remove: the disk still holds the old record, so
		// leaving the unpersisted one in memory would diverge the two until a restart.
		s.Model().AddressIndex = index

		if ok {
			s.Model().Things[model.ID] = old
		} else {
			delete(s.Model().Things, model.ID)
		}

		return nil, fmt.Errorf("state: failed to persist state of a thing with ID %s: %w", model.ID, err)
	}

	return newThingState(s, model), nil
}

// modelByID returns the record itself, not a copy: add replaces the map entry rather than
// mutating it, so a caller can hand the old record straight back to add to undo an overwrite.
func (s *state) modelByID(id string) *thingStateModel {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.Model().Things[id]
}

func (s *state) remove(id string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	old, ok := s.Model().Things[id]

	delete(s.Model().Things, id)

	if err := s.Save(); err != nil {
		// Restore on a failed write: the disk still holds the record, so keeping it in memory
		// lets the next sync retry the destroy instead of resurrecting the thing only after
		// a restart.
		if ok {
			s.Model().Things[id] = old
		}

		return fmt.Errorf("state: failed to remove state of a thing with ID %s: %w", id, err)
	}

	return nil
}

func (s *state) byID(id string) ThingState {
	s.lock.RLock()
	defer s.lock.RUnlock()

	ts, ok := s.Model().Things[id]
	if !ok {
		return nil
	}

	return newThingState(s, ts)
}

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
	// InclusionChecksum returns the checksum of the inclusion report stored in the thing state.
	InclusionChecksum() uint32
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

type thingState struct {
	state *state
	model *thingStateModel
}

func (s *thingState) ID() string {
	s.state.lock.RLock()
	defer s.state.lock.RUnlock()

	return s.model.ID
}

func (s *thingState) Address() string {
	s.state.lock.RLock()
	defer s.state.lock.RUnlock()

	return s.model.Address
}

func (s *thingState) Info(model any) error {
	s.state.lock.RLock()
	defer s.state.lock.RUnlock()

	if len(s.model.Info) == 0 {
		return nil
	}

	err := json.Unmarshal(s.model.Info, model)
	if err != nil {
		return fmt.Errorf("thing state: failed to unmarshal info of a thing with ID %s into a provided model: %w", s.model.ID, err)
	}

	return nil
}

func (s *thingState) State(model any) error {
	s.state.lock.RLock()
	defer s.state.lock.RUnlock()

	if len(s.model.State) == 0 {
		return nil
	}

	err := json.Unmarshal(s.model.State, model)
	if err != nil {
		return fmt.Errorf("thing state: failed to unmarshal state of a thing with ID %s into a provided model: %w", s.model.ID, err)
	}

	return nil
}

func (s *thingState) SetState(model any) error {
	s.state.lock.Lock()
	defer s.state.lock.Unlock()

	b, err := json.Marshal(model)
	if err != nil {
		return fmt.Errorf("thing state: failed to marshal state of a thing with ID %s from a provided model: %w", s.model.ID, err)
	}

	s.model.State = b

	err = s.state.Save()
	if err != nil {
		return fmt.Errorf("thing state: failed to persist state of a thing with ID %s: %w", s.model.ID, err)
	}

	return nil
}

func (s *thingState) InclusionChecksum() uint32 {
	s.state.lock.RLock()
	defer s.state.lock.RUnlock()

	return s.model.InclusionChecksum
}

func (s *thingState) SetInclusionChecksum(checksum uint32) error {
	s.state.lock.Lock()
	defer s.state.lock.Unlock()

	if s.model.InclusionChecksum == checksum {
		return nil
	}

	// Restored on a failed write, mirroring add and remove: the skip above is keyed on the
	// in-memory value, so leaving it set would make every retry of the same checksum a no-op and
	// the disk would never catch up until the report changed again.
	previous := s.model.InclusionChecksum
	s.model.InclusionChecksum = checksum

	err := s.state.Save()
	if err != nil {
		s.model.InclusionChecksum = previous

		return fmt.Errorf("thing state: failed to persist inclusion checksum of a thing with ID %s: %w", s.model.ID, err)
	}

	return nil
}
