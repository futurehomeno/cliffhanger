package adapter

import (
	"fmt"
	"strings"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
)

type Adapter interface {
	Name() string
	Address() string
	Services(name string) []Service
	ServiceByTopic(topic string) Service
	Things() []Thing
	ThingByAddress(address string) Thing
	ThingByTopic(topic string) Thing
	RegisterThing(thing Thing)
	UnregisterThing(address string)
	AddThing(thing Thing) error
	RemoveThing(address string) error
	RemoveAllThings() error
	SendInclusionReport(thing Thing) error
	SendExclusionReport(thing Thing) error
}

func NewAdapter(mqtt *fimpgo.MqttTransport, resourceName, resourceAddress string) Adapter {
	return &adapter{
		lock:         &sync.RWMutex{},
		mqtt:         mqtt,
		name:         resourceName,
		address:      resourceAddress,
		addressIndex: make(map[string]Thing),
		topicIndex:   make(map[string]Thing),
	}
}

type adapter struct {
	lock *sync.RWMutex
	mqtt *fimpgo.MqttTransport

	name    string
	address string

	addressIndex map[string]Thing
	topicIndex   map[string]Thing
}

func (a *adapter) Name() string {
	return a.name
}

func (a *adapter) Address() string {
	return a.name
}

func (a *adapter) Things() []Thing {
	var things []Thing

	for _, t := range a.addressIndex {
		things = append(things, t)
	}

	return things
}

func (a *adapter) Services(name string) []Service {
	var services []Service

	for _, t := range a.addressIndex {
		services = append(services, t.Services(name)...)
	}

	return services
}

func (a *adapter) ServiceByTopic(topic string) Service {
	t := a.ThingByTopic(topic)
	if t == nil {
		return nil
	}

	return t.ServiceByTopic(topic)
}

func (a *adapter) ThingByAddress(address string) Thing {
	a.lock.RLock()
	defer a.lock.RUnlock()

	return a.addressIndex[address]
}

func (a *adapter) ThingByTopic(topic string) Thing {
	a.lock.RLock()
	defer a.lock.RUnlock()

	for serviceTopic, t := range a.topicIndex {
		if strings.HasSuffix(topic, serviceTopic) {
			return t
		}
	}

	return nil
}

func (a *adapter) RegisterThing(thing Thing) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.register(thing)
}

func (a *adapter) UnregisterThing(address string) {
	a.lock.Lock()
	defer a.lock.Unlock()

	thing := a.ThingByAddress(address)
	if thing == nil {
		return
	}

	a.unregister(thing)
}

func (a *adapter) AddThing(thing Thing) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.register(thing)

	return a.SendInclusionReport(thing)
}

func (a *adapter) RemoveThing(address string) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	t := a.ThingByAddress(address)
	if t == nil {
		return nil
	}

	a.unregister(t)

	return a.SendExclusionReport(t)
}

func (a *adapter) RemoveAllThings() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	for _, t := range a.addressIndex {
		a.unregister(t)

		err := a.SendExclusionReport(t)
		if err != nil {
			return err
		}
	}

	return nil
}

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

func (a *adapter) register(thing Thing) {
	a.addressIndex[thing.Address()] = thing

	for _, topic := range thing.ServiceTopics() {
		a.topicIndex[topic] = thing
	}
}

func (a *adapter) unregister(thing Thing) {
	delete(a.addressIndex, thing.Address())

	for _, topic := range thing.ServiceTopics() {
		delete(a.topicIndex, topic)
	}
}
