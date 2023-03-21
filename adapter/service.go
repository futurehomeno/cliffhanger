package adapter

import (
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
func NewService(adapter Adapter, specification *fimptype.Service) Service {
	return &service{
		adapter:       adapter,
		specification: specification,
	}
}

// Service is a private implementation of a FIMP service.
type service struct {
	adapter       Adapter
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
	return s.adapter.publishServiceMessage(s, message)
}
