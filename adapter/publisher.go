package adapter

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/event"
)

type Publisher interface {
	ThingPublisher

	PublishAdapterMessage(message *fimpgo.FimpMessage) error
}

// ThingPublisher is an interface representing a FIMP thing publisher.
type ThingPublisher interface {
	ServicePublisher

	PublishThingMessage(thing Thing, message *fimpgo.FimpMessage) error
}

// ServicePublisher is an interface representing a FIMP service publisher.
type ServicePublisher interface {
	PublishServiceMessage(service Service, message *fimpgo.FimpMessage) error
	PublishServiceEvent(service Service, payload *ServiceEvent)
}

func NewPublisher(eventManager event.Manager, mqtt *fimpgo.MqttTransport, adapterName, adapterAddress string) Publisher {
	return &publisher{
		eventManager:   eventManager,
		mqtt:           mqtt,
		adapterName:    adapterName,
		adapterAddress: adapterAddress,
	}
}

type publisher struct {
	eventManager event.Manager
	mqtt         *fimpgo.MqttTransport

	adapterName    string
	adapterAddress string
}

func (p *publisher) PublishServiceMessage(service Service, message *fimpgo.FimpMessage) error {
	address, err := fimpgo.NewAddressFromString(service.Topic())
	if err != nil {
		return fmt.Errorf("adapter: failed to parse a service topic %s: %w", service.Topic(), err)
	}

	address.MsgType = fimpgo.MsgTypeEvt
	message.Service = service.Name()

	err = p.mqtt.Publish(address, message)
	if err != nil {
		return fmt.Errorf("adapter: failed to publish a service report: %w", err)
	}

	return nil
}

func (p *publisher) PublishThingMessage(thing Thing, message *fimpgo.FimpMessage) error {
	address := &fimpgo.Address{
		MsgType:         fimpgo.MsgTypeEvt,
		ResourceType:    fimpgo.ResourceTypeAdapter,
		ResourceName:    p.adapterName,
		ResourceAddress: p.adapterAddress,
	}

	message.Service = p.adapterName

	err := p.mqtt.Publish(address, message)
	if err != nil {
		return fmt.Errorf("adapter: failed to publish a thing with address %s report: %w", thing.Address(), err)
	}

	return nil
}

func (p *publisher) PublishServiceEvent(service Service, payload *ServiceEvent) {
	p.eventManager.Publish(&event.Event{
		Domain:  ServiceEventDomain(service.Name()),
		Payload: payload,
	})
}

func (p *publisher) PublishAdapterMessage(message *fimpgo.FimpMessage) error {
	address := &fimpgo.Address{
		MsgType:         fimpgo.MsgTypeEvt,
		ResourceType:    fimpgo.ResourceTypeAdapter,
		ResourceName:    p.adapterName,
		ResourceAddress: p.adapterAddress,
	}

	message.Service = p.adapterName

	err := p.mqtt.Publish(address, message)
	if err != nil {
		return fmt.Errorf("adapter: failed to publish an adapter report: %w", err)
	}

	return nil
}
