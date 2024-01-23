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
	PublishThingEvent(thingEvent ThingEvent)
}

// ServicePublisher is an interface representing a FIMP service publisher.
type ServicePublisher interface {
	PublishServiceMessage(service Service, message *fimpgo.FimpMessage) error
	PublishServiceEvent(service Service, payload ServiceEvent)
}

func NewPublisher(mqtt *fimpgo.MqttTransport, eventManager event.Manager, adapterName, adapterAddress string) Publisher {
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

// PublishServiceEvent publishes an event to the local event manager.
func (p *publisher) PublishServiceEvent(service Service, serviceEvent ServiceEvent) {
	serviceEvent.setEvent(event.New(EventDomainAdapterService, service.Name()))
	serviceEvent.setAddress(service.Topic())
	serviceEvent.setServiceName(service.Name())

	p.eventManager.Publish(serviceEvent)
}

// PublishThingEvent publishes an event to the local event manager.
func (p *publisher) PublishThingEvent(thingEvent ThingEvent) {
	p.eventManager.Publish(thingEvent)
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
