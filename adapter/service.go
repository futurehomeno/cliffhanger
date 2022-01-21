package adapter

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
)

// Service is an interface representing a FIMP service.
type Service interface {
	// Name returns service name.
	Name() string
	// Topic returns topic under which service should be listening for commands.
	Topic() string
	// Specification returns service FIMP specification.
	Specification() *fimptype.Service
	// SendMessage sends a message from the service with provided contents.
	SendMessage(message *fimpgo.FimpMessage) error
}

// NewService creates instance of a FIMP service.
func NewService(mqtt *fimpgo.MqttTransport, specification *fimptype.Service) Service {
	return &service{
		mqtt:          mqtt,
		specification: specification,
	}
}

// Service is a private implementation of a FIMP service.
type service struct {
	mqtt          *fimpgo.MqttTransport
	specification *fimptype.Service
}

// Name returns service name.
func (s *service) Name() string {
	return s.specification.Name
}

// Topic returns topic under which service should be listening for commands.
func (s *service) Topic() string {
	return s.specification.Address
}

// Specification returns service FIMP specification.
func (s *service) Specification() *fimptype.Service {
	return s.specification
}

// SendMessage sends a message from the service with provided contents.
func (s *service) SendMessage(message *fimpgo.FimpMessage) error {
	address, err := fimpgo.NewAddressFromString(s.Topic())
	if err != nil {
		return fmt.Errorf("service: failed to parse service topic %s: %w", s.Topic(), err)
	}

	address.MsgType = fimpgo.MsgTypeEvt
	message.Service = s.Name()

	err = s.mqtt.Publish(address, message)
	if err != nil {
		return fmt.Errorf("service: failed to publish report: %w", err)
	}

	return nil
}
