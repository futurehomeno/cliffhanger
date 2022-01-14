package adapter

import (
	"fmt"
	"strings"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
)

type Thing interface {
	GetInclusionReport() *fimptype.ThingInclusionReport
	GetAddress() string
	GetServiceTopics() []string
}

type Adapter interface {
	GetName() string
	GetByAddress(address string) Thing
	GetByTopic(topic string) Thing
	Register(thing Thing)
	Unregister(address string)
	Add(thing Thing) error
	Remove(address string) error
	RemoveAll() error
	SendInclusionReport(thing Thing) error
	SendExclusionReport(thing Thing) error
}

func NewAdapter(mqtt *fimpgo.MqttTransport, serviceName, instanceID string) Adapter {
	return &adapter{
		lock:         &sync.RWMutex{},
		mqtt:         mqtt,
		name:         serviceName,
		instanceID:   instanceID,
		addressIndex: nil,
		topicIndex:   nil,
	}
}

type adapter struct {
	lock *sync.RWMutex
	mqtt *fimpgo.MqttTransport

	name       string
	instanceID string

	addressIndex map[string]Thing
	topicIndex   map[string]Thing
}

func (a *adapter) GetName() string {
	return a.name
}

func (a *adapter) GetByAddress(address string) Thing {
	a.lock.RLock()
	defer a.lock.RUnlock()

	return a.addressIndex[address]
}

func (a *adapter) GetByTopic(topic string) Thing {
	a.lock.RLock()
	defer a.lock.RUnlock()

	for serviceTopic, thing := range a.topicIndex {
		if strings.HasSuffix(topic, serviceTopic) {
			return thing
		}
	}

	return nil
}

func (a *adapter) Register(thing Thing) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.register(thing)
}

func (a *adapter) Unregister(address string) {
	a.lock.Lock()
	defer a.lock.Unlock()

	thing := a.GetByAddress(address)
	if thing == nil {
		return
	}

	a.unregister(thing)
}

func (a *adapter) Add(thing Thing) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.register(thing)

	return a.SendInclusionReport(thing)
}

func (a *adapter) Remove(address string) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	thing := a.GetByAddress(address)
	if thing == nil {
		return nil
	}

	a.unregister(thing)

	return a.SendExclusionReport(thing)
}

func (a *adapter) RemoveAll() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	for _, thing := range a.addressIndex {
		a.unregister(thing)

		err := a.SendExclusionReport(thing)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *adapter) SendInclusionReport(thing Thing) error {
	report := thing.GetInclusionReport()

	addr := &fimpgo.Address{
		MsgType:         fimpgo.MsgTypeEvt,
		ResourceType:    fimpgo.ResourceTypeAdapter,
		ResourceName:    a.name,
		ResourceAddress: a.instanceID,
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
		Address: thing.GetAddress(),
	}

	addr := &fimpgo.Address{
		MsgType:         fimpgo.MsgTypeEvt,
		ResourceType:    fimpgo.ResourceTypeAdapter,
		ResourceName:    a.name,
		ResourceAddress: a.instanceID,
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
	a.addressIndex[thing.GetAddress()] = thing

	for _, topic := range thing.GetServiceTopics() {
		a.topicIndex[topic] = thing
	}
}

func (a *adapter) unregister(thing Thing) {
	delete(a.addressIndex, thing.GetAddress())

	for _, topic := range thing.GetServiceTopics() {
		delete(a.topicIndex, topic)
	}
}
