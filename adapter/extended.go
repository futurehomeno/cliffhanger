package adapter

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/event"
)

// Adapter is an interface representing an extended and stateful device adapter.
// It acts as a manager extension of an adapter, abstracting away logic for management of devices.
type Adapter interface {
	baseAdapter

	// ThingByID returns a thing based on its ID. Returns nil if thing was not found.
	ThingByID(id string) Thing
	// ExchangeID returns address of a thing with a given ID.
	ExchangeID(id string) (address string, ok bool)
	// ExchangeAddress returns ID of a thing with a given address.
	ExchangeAddress(address string) (id string, ok bool)
	// InitializeThings reads all things stored in a persistent state and registers them.
	// This method should be called once during adapter booting.
	InitializeThings() error
	// EnsureThings creates and destroys things based on provided map of IDs and custom information objects.
	EnsureThings(idsAndInfo map[string]interface{}) error
	// CreateThing creates thing and adds it to the adapter.
	CreateThing(id string, info interface{}) error
	// DestroyThing destroys thing and removes it from the adapter.
	DestroyThing(id string) error
	// DestroyAllThings destroys all things and removes them from the adapter.
	DestroyAllThings() error
}

// NewAdapter creates new instance of an extended adapter.
func NewAdapter(
	mqtt *fimpgo.MqttTransport,
	eventManager event.Manager,
	factory ThingFactory,
	state State,
	resourceName, resourceAddress string,
) Adapter {
	return &extendedAdapter{
		baseAdapter: newBaseAdapter(mqtt, resourceName, resourceAddress),
		factory:     factory,
		state:       state,
		mqtt:        mqtt,
		lock:        &sync.RWMutex{},
	}
}

// extendedAdapter is a private implementation of an extended adapter service.
type extendedAdapter struct {
	baseAdapter

	state   State
	factory ThingFactory
	mqtt    *fimpgo.MqttTransport
	lock    *sync.RWMutex
}

// ThingByID returns a thing based on its ID. Returns nil if thing was not found.
func (a *extendedAdapter) ThingByID(id string) Thing {
	a.lock.RLock()
	defer a.lock.RUnlock()

	ts := a.state.byID(id)
	if ts == nil {
		return nil
	}

	return a.ThingByAddress(ts.Address())
}

// ExchangeID returns address of a thing with a given ID.
func (a *extendedAdapter) ExchangeID(id string) (address string, ok bool) {
	a.lock.RLock()
	defer a.lock.RUnlock()

	ts := a.state.byID(id)
	if ts == nil {
		return "", false
	}

	return ts.Address(), true
}

// ExchangeAddress returns ID of a thing with a given address.
func (a *extendedAdapter) ExchangeAddress(address string) (id string, ok bool) {
	a.lock.RLock()
	defer a.lock.RUnlock()

	ts := a.state.byAddress(address)
	if ts == nil {
		return "", false
	}

	return ts.ID(), true
}

// InitializeThings reads all things stored in a persistent state and registers them.
// This method should be called once during adapter booting.
func (a *extendedAdapter) InitializeThings() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	for _, ts := range a.state.all() {
		t, err := a.factory.Create(a, ts)
		if err != nil {
			return fmt.Errorf("adapter: failed to create thing with ID %s: %w", ts.ID(), err)
		}

		a.RegisterThing(t)

		_, err = t.SendInclusionReport(false)
		if err != nil {
			return fmt.Errorf("adapter: failed to refresh inclusion report for thing with ID %s: %w", ts.ID(), err)
		}
	}

	return nil
}

// EnsureThings creates and destroys things based on provided map of IDs and custom information objects.
func (a *extendedAdapter) EnsureThings(idsAndInfo map[string]interface{}) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	var toRemove []string

	thingStates := a.state.all()

	for _, ts := range thingStates {
		_, ok := idsAndInfo[ts.ID()]
		if !ok {
			toRemove = append(toRemove, ts.ID())

			continue
		}

		delete(idsAndInfo, ts.ID())
	}

	for _, id := range toRemove {
		err := a.destroyThing(id)
		if err != nil {
			return fmt.Errorf("adapter: failed to destroy thing with ID %s: %w", id, err)
		}
	}

	for id, info := range idsAndInfo {
		err := a.createThing(id, info)
		if err != nil {
			return fmt.Errorf("adapter: failed to create thing with ID %s: %w", id, err)
		}
	}

	return nil
}

// CreateThing creates thing and adds it to the adapter.
func (a *extendedAdapter) CreateThing(id string, info interface{}) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if err := a.createThing(id, info); err != nil {
		return fmt.Errorf("adapter: failed to create thing with ID %s: %w", id, err)
	}

	return nil
}

// DestroyThing destroys thing and removes it from the adapter.
func (a *extendedAdapter) DestroyThing(id string) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if err := a.destroyThing(id); err != nil {
		return fmt.Errorf("adapter: failed to destroy thing with ID %s: %w", id, err)
	}

	return nil
}

// DestroyAllThings destroys all things and removes them from the adapter.
func (a *extendedAdapter) DestroyAllThings() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	for _, ts := range a.state.all() {
		err := a.destroyThing(ts.ID())
		if err != nil {
			return fmt.Errorf("adapter: failed to destroy thing with ID %s: %w", ts.ID(), err)
		}
	}

	return nil
}

// createThing utilizes factory to create a thing, persists it in the state and adds to the adapter.
func (a *extendedAdapter) createThing(id string, info interface{}) error {
	ts, err := a.createThingState(id, info)
	if err != nil {
		return fmt.Errorf("adapter: failed to create state for thing with ID %s: %w", id, err)
	}

	t, err := a.factory.Create(a, ts)
	if err != nil {
		return fmt.Errorf("adapter: failed to create thing with ID %s: %w", id, err)
	}

	err = a.AddThing(t)
	if err != nil {
		return fmt.Errorf("adapter: failed to add thing with ID %s to the adapter: %w", id, err)
	}

	return nil
}

// createThingState creates new state of a thing and acquires a new address for it.
func (a *extendedAdapter) createThingState(id string, info interface{}) (ThingState, error) {
	address, err := a.state.acquireAddress()
	if err != nil {
		return nil, fmt.Errorf("adapter: failed to accquire a new address for thing with ID %s: %w", id, err)
	}

	model := &thingStateModel{
		ID:      id,
		Address: address,
	}

	if info != nil {
		b, err := json.Marshal(info)
		if err != nil {
			return nil, fmt.Errorf("adapter: failed to marshal additional information associated with thing with ID %s: %w", id, err)
		}

		model.Info = b
	}

	ts, err := a.state.add(model)
	if err != nil {
		return nil, fmt.Errorf("adapter: failed to persist state of thing with ID %s: %w", id, err)
	}

	return ts, nil
}

// destroyThing removes thing from an adapter and deletes it from the state.
func (a *extendedAdapter) destroyThing(id string) error {
	ts := a.state.byID(id)
	if ts == nil {
		return nil
	}

	err := a.RemoveThing(ts.Address())
	if err != nil {
		return fmt.Errorf("adapter: failed to remove thing with ID %s from the adapter: %w", id, err)
	}

	err = a.state.remove(ts.ID())
	if err != nil {
		return fmt.Errorf("adapter: failed to remove thing with ID %s from persistent state: %w", id, err)
	}

	return nil
}
