package adapter

import (
	"fmt"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
)

// baseAdapter is an interface representing a device adapter.
type baseAdapter interface {
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
	// SendExclusionReport sends exclusion report for a specific thing.
	SendExclusionReport(address string) error
	SendAllNodesReport() error

	publishAdapterMessage(message *fimpgo.FimpMessage) error
	publishThingMessage(thing Thing, message *fimpgo.FimpMessage) error
	publishServiceMessage(service Service, message *fimpgo.FimpMessage) error
}

// newBaseAdapter creates an instance of a device adapter.
func newBaseAdapter(mqtt *fimpgo.MqttTransport, resourceName, resourceAddress string) baseAdapter {
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

	thing.Connect()
}

// UnregisterThing unregisters thing from the adapter without sending an exclusion report.
func (a *adapter) UnregisterThing(address string) {
	t := a.ThingByAddress(address)
	if t == nil {
		return
	}

	a.lock.Lock()
	defer a.lock.Unlock()

	t.Disconnect()

	delete(a.things, t.Address())
}

// UnregisterAllThings unregisters all things from the adapter without sending an exclusion report.
func (a *adapter) UnregisterAllThings() {
	a.lock.Lock()
	defer a.lock.Unlock()

	for _, t := range a.things {
		t.Disconnect()
	}

	a.things = make(map[string]Thing)
}

// AddThing registers thing and sends an inclusion report. Useful when configuring adapter for the first time.
func (a *adapter) AddThing(thing Thing) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.things[thing.Address()] = thing

	defer thing.Connect()

	if _, err := thing.SendInclusionReport(true); err != nil {
		return fmt.Errorf("adapter: failed to send the inclusion report for thing with address %s: %w", thing.Address(), err)
	}

	return nil
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

	t.Disconnect()

	if err := a.SendExclusionReport(t.Address()); err != nil {
		return fmt.Errorf("adapter: failed to send the exclusion report for thing with address %s: %w", t.Address(), err)
	}

	return nil
}

// RemoveAllThings unregisters all things and sends exclusion reports. Useful when uninstalling or resetting adapter.
func (a *adapter) RemoveAllThings() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	for _, t := range a.things {
		delete(a.things, t.Address())

		t.Disconnect()

		err := a.SendExclusionReport(t.Address())
		if err != nil {
			return fmt.Errorf("adapter: failed to send the exclusion report for thing with address %s: %w", t.Address(), err)
		}
	}

	return nil
}

// SendExclusionReport sends exclusion report for a specific thing.
func (a *adapter) SendExclusionReport(address string) error {
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

	err := a.mqtt.Publish(a.eventAddress(), msg)
	if err != nil {
		return fmt.Errorf("adapter: failed to publish the exclusion report for thing with address %s: %w", address, err)
	}

	return nil
}

func (a *adapter) SendAllNodesReport() error {
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

	return a.publishAdapterMessage(msg)
}

func (a *adapter) eventAddress() *fimpgo.Address {
	return &fimpgo.Address{
		MsgType:         fimpgo.MsgTypeEvt,
		ResourceType:    fimpgo.ResourceTypeAdapter,
		ResourceName:    a.Name(),
		ResourceAddress: a.Address(),
	}
}

func (a *adapter) publishAdapterMessage(message *fimpgo.FimpMessage) error {
	address := &fimpgo.Address{
		MsgType:         fimpgo.MsgTypeEvt,
		ResourceType:    fimpgo.ResourceTypeAdapter,
		ResourceName:    a.Name(),
		ResourceAddress: a.Address(),
	}

	message.Service = a.Name()

	err := a.mqtt.Publish(address, message)
	if err != nil {
		return fmt.Errorf("adapter: failed to publish adapter report: %w", err)
	}

	return nil
}

func (a *adapter) publishThingMessage(_ Thing, message *fimpgo.FimpMessage) error {
	address := &fimpgo.Address{
		MsgType:         fimpgo.MsgTypeEvt,
		ResourceType:    fimpgo.ResourceTypeAdapter,
		ResourceName:    a.Name(),
		ResourceAddress: a.Address(),
	}

	message.Service = a.Name()

	err := a.mqtt.Publish(address, message)
	if err != nil {
		return fmt.Errorf("adapter: failed to publish a thing report: %w", err)
	}

	return nil
}

func (a *adapter) publishServiceMessage(service Service, message *fimpgo.FimpMessage) error {
	address, err := fimpgo.NewAddressFromString(service.Topic())
	if err != nil {
		return fmt.Errorf("adapter: failed to parse a service topic %s: %w", service.Topic(), err)
	}

	address.MsgType = fimpgo.MsgTypeEvt
	message.Service = service.Name()

	err = a.mqtt.Publish(address, message)
	if err != nil {
		return fmt.Errorf("adapter: failed to publish a service report: %w", err)
	}

	return nil
}
