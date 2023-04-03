package adapter

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
)

type Publisher interface {
	PublishServiceMessage(service Service, message *fimpgo.FimpMessage) error
	PublishThingMessage(thing Thing, message *fimpgo.FimpMessage) error
	PublishAdapterMessage(message *fimpgo.FimpMessage) error
}

func NewPublisher(mqtt *fimpgo.MqttTransport, adapterName, adapterAddress string) Publisher {
	return &publisher{
		mqtt:           mqtt,
		adapterName:    adapterName,
		adapterAddress: adapterAddress,
	}
}

type publisher struct {
	mqtt *fimpgo.MqttTransport

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
