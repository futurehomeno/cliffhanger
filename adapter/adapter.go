package adapter

import (
	"fmt"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
)

// Adapter is an interface representing a device adapter.
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
	// RegisterThing registers thing with the adapter without sending an inclusion report. Useful when restarting adapter.
	RegisterThing(thing Thing)
	// UnregisterThing unregisters thing from the adapter without sending an exclusion report.
	UnregisterThing(address string)
	// UnregisterAllThings unregisters all things from the adapter without sending an exclusion report.
	UnregisterAllThings()
	// AddThing registers thing and sends an inclusion report. Useful when configuring adapter for the first time.
	AddThing(thing Thing) error
	// RemoveThing unregisters thing and sends exclusion report.
	RemoveThing(address string) error
	// RemoveAllThings unregisters all things and sends exclusion reports. Useful when uninstalling or resetting adapter.
	RemoveAllThings() error
	// SendInclusionReport sends inclusion report for a specific thing.
	SendInclusionReport(thing Thing) error
	// SendExclusionReport sends exclusion report for a specific thing.
	SendExclusionReport(thing Thing) error
}

// NewAdapter creates an instance of a device adapter.
func NewAdapter(mqtt *fimpgo.MqttTransport, resourceName, resourceAddress string) Adapter {
	return &adapter{
		lock:    &sync.RWMutex{},
		mqtt:    mqtt,
		name:    resourceName,
		address: resourceAddress,
		things:  make(map[string]Thing),
	}
}

// adapter is a private implementation of a device adapter.
type adapter struct {
	lock *sync.RWMutex
	mqtt *fimpgo.MqttTransport

	name    string
	address string

	things map[string]Thing
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
	var services []Service

	for _, t := range a.things {
		services = append(services, t.Services(name)...)
	}

	return services
}

// ServiceByTopic returns a service based on its topic. Returns nil if service was not found.
func (a *adapter) ServiceByTopic(topic string) Service {
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
	var things []Thing

	for _, t := range a.things {
		things = append(things, t)
	}

	return things
}

// ThingByAddress returns a thing based on its address. Returns nil if thing was not found.
func (a *adapter) ThingByAddress(address string) Thing {
	a.lock.RLock()
	defer a.lock.RUnlock()

	return a.things[address]
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

// RegisterThing registers thing with the adapter without sending an inclusion report. Useful when restarting adapter.
func (a *adapter) RegisterThing(thing Thing) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.things[thing.Address()] = thing
}

// UnregisterThing unregisters thing from the adapter without sending an exclusion report.
func (a *adapter) UnregisterThing(address string) {
	t := a.ThingByAddress(address)
	if t == nil {
		return
	}

	a.lock.Lock()
	defer a.lock.Unlock()

	delete(a.things, t.Address())
}

// UnregisterAllThings unregisters all things from the adapter without sending an exclusion report.
func (a *adapter) UnregisterAllThings() {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.things = make(map[string]Thing)
}

// AddThing registers thing and sends an inclusion report. Useful when configuring adapter for the first time.
func (a *adapter) AddThing(thing Thing) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.things[thing.Address()] = thing

	return a.SendInclusionReport(thing)
}

// RemoveThing unregisters thing and sends exclusion report.
func (a *adapter) RemoveThing(address string) error {
	t := a.ThingByAddress(address)
	if t == nil {
		return nil
	}

	a.lock.Lock()
	defer a.lock.Unlock()

	delete(a.things, t.Address())

	return a.SendExclusionReport(t)
}

// RemoveAllThings unregisters all things and sends exclusion reports. Useful when uninstalling or resetting adapter.
func (a *adapter) RemoveAllThings() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	for _, t := range a.things {
		delete(a.things, t.Address())

		err := a.SendExclusionReport(t)
		if err != nil {
			return err
		}
	}

	return nil
}

// SendInclusionReport sends inclusion report for a specific thing.
func (a *adapter) SendInclusionReport(thing Thing) error {
	report := thing.InclusionReport()

	addr := &fimpgo.Address{
		MsgType:         fimpgo.MsgTypeEvt,
		ResourceType:    fimpgo.ResourceTypeAdapter,
		ResourceName:    a.Name(),
		ResourceAddress: a.Address(),
	}

	msg := fimpgo.NewObjectMessage(
		EvtThingInclusionReport,
		a.name,
		report,
		nil,
		nil,
		nil,
	)

	err := a.mqtt.Publish(addr, msg)
	if err != nil {
		return fmt.Errorf("adapter: failed to publish the inclusion report")
	}

	return nil
}

// SendExclusionReport sends exclusion report for a specific thing.
func (a *adapter) SendExclusionReport(thing Thing) error {
	report := fimptype.ThingExclusionReport{
		Address: thing.Address(),
	}

	addr := &fimpgo.Address{
		MsgType:         fimpgo.MsgTypeEvt,
		ResourceType:    fimpgo.ResourceTypeAdapter,
		ResourceName:    a.Name(),
		ResourceAddress: a.Address(),
	}

	msg := fimpgo.NewObjectMessage(
		EvtThingExclusionReport,
		a.name,
		report,
		nil,
		nil,
		nil,
	)

	err := a.mqtt.Publish(addr, msg)
	if err != nil {
		return fmt.Errorf("adapter: failed to publish the exclusion report")
	}

	return nil
}
