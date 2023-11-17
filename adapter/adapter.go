package adapter

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
)

// Adapter is an interface representing an stateful device adapter.
// It acts as a manager abstracting business logic for management of devices.
type Adapter interface {
	// Name returns name of the adapter.
	Name() string
	// Address returns an address of the adapter.
	Address() string
	// Services returns all services from all things that match the provided name. If empty all services are returned.
	Services(name string) []Service
	// ServiceByTopic returns a service based on its topic. Returns nil if service was not found.
	ServiceByTopic(topic string) Service
	// Things returns all things.
	Things() []Thing
	// ThingByAddress returns a thing based on its address. Returns nil if thing was not found.
	ThingByAddress(address string) Thing
	// ThingByTopic returns a thing based on topic of one of its services. Returns nil if thing was not found.
	ThingByTopic(topic string) Thing
	// ThingByID returns a thing based on its internal ID. Returns nil if thing was not found.
	ThingByID(id string) Thing
	// ExchangeID returns address of a thing with a given ID.
	ExchangeID(id string) (address string, ok bool)
	// ExchangeAddress returns ID of a thing with a given address.
	ExchangeAddress(address string) (id string, ok bool)
	// IsInitialized returns true if adapter is initialized.
	IsInitialized() bool
	// InitializeThings reads all things stored in a persistent state and registers them.
	// This method should be called once during adapter booting.
	InitializeThings() error
	// EnsureThings creates and destroys things based on provided map of IDs and custom information objects.
	EnsureThings(seeds ThingSeeds) error
	// CreateThing creates thing and adds it to the adapter.
	CreateThing(seed *ThingSeed) error
	// DestroyThingByID destroys thing and removes it from the adapter.
	DestroyThingByID(id string) error
	// DestroyThingByAddress destroys thing and removes it from the adapter.
	DestroyThingByAddress(address string) error
	// DestroyAllThings destroys all things and removes them from the adapter.
	DestroyAllThings() error
	// SendConnectivityReport sends connectivity report for all things registered within the adapter.
	SendConnectivityReport() error
}

// NewAdapter creates new instance of an extended adapter.
func NewAdapter(
	mqtt *fimpgo.MqttTransport,
	factory ThingFactory,
	state State,
	resourceName, resourceAddress string,
) Adapter {
	return &adapter{
		name:      resourceName,
		address:   resourceAddress,
		things:    make(map[string]Thing),
		factory:   factory,
		state:     state,
		publisher: NewPublisher(mqtt, resourceName, resourceAddress),
		lock:      &sync.RWMutex{},
	}
}

// adapter is a private implementation of an adapter service.
type adapter struct {
	publisher Publisher
	state     State
	factory   ThingFactory

	name        string
	address     string
	things      map[string]Thing
	initialized bool
	lock        *sync.RWMutex
}

// Name returns name of the adapter.
func (a *adapter) Name() string {
	return a.name
}

// Address returns an address of the adapter.
func (a *adapter) Address() string {
	return a.address
}

// Services returns all services from all things that match the provided name. If empty all services are returned.
func (a *adapter) Services(name string) []Service {
	a.lock.RLock()
	defer a.lock.RUnlock()

	var services []Service

	for _, t := range a.things {
		services = append(services, t.Services(name)...)
	}

	return services
}

// ServiceByTopic returns a service based on its topic. Returns nil if service was not found.
func (a *adapter) ServiceByTopic(topic string) Service {
	a.lock.RLock()
	defer a.lock.RUnlock()

	for _, t := range a.things {
		s := t.ServiceByTopic(topic)
		if s != nil {
			return s
		}
	}

	return nil
}

// Things returns all things.
func (a *adapter) Things() []Thing {
	a.lock.RLock()
	defer a.lock.RUnlock()

	var things []Thing

	for _, t := range a.things {
		things = append(things, t)
	}

	return things
}

// ThingByTopic returns a thing based on topic of one of its services. Returns nil if thing was not found.
func (a *adapter) ThingByTopic(topic string) Thing {
	a.lock.RLock()
	defer a.lock.RUnlock()

	for _, t := range a.things {
		s := t.ServiceByTopic(topic)
		if s != nil {
			return t
		}
	}

	return nil
}

// ThingByAddress returns a thing based on its address. Returns nil if thing was not found.
func (a *adapter) ThingByAddress(address string) Thing {
	a.lock.RLock()
	defer a.lock.RUnlock()

	return a.things[address]
}

// ThingByID returns a thing based on its ID. Returns nil if thing was not found.
func (a *adapter) ThingByID(id string) Thing {
	a.lock.RLock()
	defer a.lock.RUnlock()

	ts := a.state.byID(id)
	if ts == nil {
		return nil
	}

	return a.things[ts.Address()]
}

// ExchangeID returns address of a thing with a given ID.
func (a *adapter) ExchangeID(id string) (address string, ok bool) {
	a.lock.RLock()
	defer a.lock.RUnlock()

	ts := a.state.byID(id)
	if ts == nil {
		return "", false
	}

	return ts.Address(), true
}

// ExchangeAddress returns ID of a thing with a given address.
func (a *adapter) ExchangeAddress(address string) (id string, ok bool) {
	a.lock.RLock()
	defer a.lock.RUnlock()

	ts := a.state.byAddress(address)
	if ts == nil {
		return "", false
	}

	return ts.ID(), true
}

// IsInitialized returns true if adapter is initialized.
func (a *adapter) IsInitialized() bool {
	a.lock.RLock()
	defer a.lock.RUnlock()

	return a.initialized
}

// InitializeThings reads all things stored in a persistent state and registers them.
// This method should be called once during adapter booting.
func (a *adapter) InitializeThings() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.initialized {
		return nil
	}

	var things []Thing

	for _, ts := range a.state.all() {
		t, err := a.factory.Create(a, a.publisher, ts)
		if err != nil {
			return fmt.Errorf("adapter: failed to create thing with address %s: %w", ts.Address(), err)
		}

		_, err = t.SendInclusionReport(false)
		if err != nil {
			return fmt.Errorf("adapter: failed to send inclusion report for thing with address %s: %w", ts.Address(), err)
		}

		things = append(things, t)
	}

	for _, t := range things {
		a.registerThing(t)
	}

	a.initialized = true

	return nil
}

// EnsureThings creates and destroys things based on provided map of IDs and custom information objects.
func (a *adapter) EnsureThings(seeds ThingSeeds) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	var addressesToRemove []string

	thingStates := a.state.all()

	for _, ts := range thingStates {
		if !seeds.Contains(ts.ID()) {
			addressesToRemove = append(addressesToRemove, ts.Address())

			continue
		}

		seeds = seeds.Without(ts.ID())
	}

	for _, address := range addressesToRemove {
		err := a.destroyThing(address)
		if err != nil {
			return fmt.Errorf("adapter: failed to destroy thing with address %s: %w", address, err)
		}
	}

	for _, seed := range seeds {
		err := a.createThing(seed)
		if err != nil {
			return fmt.Errorf("adapter: failed to create thing with ID %s: %w", seed.ID, err)
		}
	}

	return nil
}

// CreateThing creates thing and adds it to the adapter.
func (a *adapter) CreateThing(seed *ThingSeed) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if err := a.createThing(seed); err != nil {
		return fmt.Errorf("adapter: failed to create thing with ID %s: %w", seed.ID, err)
	}

	return nil
}

// DestroyThingByID destroys thing and removes it from the adapter.
func (a *adapter) DestroyThingByID(id string) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	ts := a.state.byID(id)
	if ts == nil {
		return nil
	}

	return a.destroyThing(ts.Address())
}

// DestroyThingByAddress destroys thing and removes it from the adapter.
func (a *adapter) DestroyThingByAddress(address string) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	return a.destroyThing(address)
}

// DestroyAllThings destroys all things and removes them from the adapter.
func (a *adapter) DestroyAllThings() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	for _, ts := range a.state.all() {
		err := a.destroyThing(ts.Address())
		if err != nil {
			return fmt.Errorf("adapter: failed to destroy thing with ID %s: %w", ts.ID(), err)
		}
	}

	a.things = make(map[string]Thing)

	return nil
}

func (a *adapter) SendConnectivityReport() error {
	var connectivityReports ConnectivityReports

	for _, t := range a.Things() {
		connectivityReports = append(connectivityReports, t.ConnectivityReport())
	}

	msg := fimpgo.NewObjectMessage(
		EvtNetworkAllNodesReport,
		a.name,
		connectivityReports,
		nil,
		nil,
		nil,
	)

	return a.publisher.PublishAdapterMessage(msg)
}

// registerThing registers thing in the adapter but does not send an inclusion report for it.
func (a *adapter) registerThing(t Thing) {
	a.things[t.Address()] = t

	t.Connect()
}

// registerThing registers thing in the adapter but does not send an inclusion report for it.
func (a *adapter) unregisterThing(t Thing) {
	delete(a.things, t.Address())

	t.Disconnect()
}

// createThing utilizes factory to create a thing, persists it in the state and adds to the adapter.
func (a *adapter) createThing(seed *ThingSeed) error {
	ts, err := a.createThingState(seed)
	if err != nil {
		return fmt.Errorf("adapter: failed to create state for thing with ID %s: %w", seed.ID, err)
	}

	t, err := a.factory.Create(a, a.publisher, ts)
	if err != nil {
		return fmt.Errorf("adapter: failed to create thing with ID %s: %w", seed.ID, err)
	}

	_, err = t.SendInclusionReport(true)
	if err != nil {
		return fmt.Errorf("adapter: failed to add thing with ID %s to the adapter: %w", seed.ID, err)
	}

	a.registerThing(t)

	return nil
}

// createThingState creates new state of a thing and acquires a new address for it.
func (a *adapter) createThingState(seed *ThingSeed) (ThingState, error) {
	var err error

	address := seed.CustomAddress
	if address == "" {
		address, err = a.state.acquireAddress()
		if err != nil {
			return nil, fmt.Errorf("adapter: failed to accquire a new address for thing with ID %s: %w", seed.ID, err)
		}
	}

	model := &thingStateModel{
		ID:      seed.ID,
		Address: address,
	}

	if seed.Info != nil {
		b, err := json.Marshal(seed.Info)
		if err != nil {
			return nil, fmt.Errorf("adapter: failed to marshal additional information associated with thing with ID %s: %w", seed.ID, err)
		}

		model.Info = b
	}

	ts, err := a.state.add(model)
	if err != nil {
		return nil, fmt.Errorf("adapter: failed to persist state of thing with ID %s: %w", seed.ID, err)
	}

	return ts, nil
}

func (a *adapter) destroyThing(address string) error {
	var err error

	ts := a.state.byAddress(address)
	if ts != nil {
		err = a.state.remove(ts.ID())
		if err != nil {
			return fmt.Errorf("adapter: failed to remove state for thing with ID %s: %w", ts.ID(), err)
		}
	}

	t, ok := a.things[address]
	if ok {
		a.unregisterThing(t)
	}

	err = a.sendExclusionReport(address)
	if err != nil {
		return fmt.Errorf("adapter: failed to send exclusion report for thing with address %s: %w", ts.Address(), err)
	}

	return nil
}

// SendExclusionReport sends exclusion report for a specific thing.
func (a *adapter) sendExclusionReport(address string) error {
	report := fimptype.ThingExclusionReport{
		Address: address,
	}

	msg := fimpgo.NewObjectMessage(
		EvtThingExclusionReport,
		a.name,
		report,
		nil,
		nil,
		nil,
	)

	return a.publisher.PublishAdapterMessage(msg)
}
