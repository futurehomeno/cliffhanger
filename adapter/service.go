package adapter

import (
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/task"
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

// ServiceRegistry is an interface representing a service registry.
type ServiceRegistry interface {
	// Services returns all services from all things that match the provided name. If empty all services are returned.
	Services(name string) []Service
	// ServiceByTopic returns a service based on its topic. Returns nil if service was not found.
	ServiceByTopic(topic string) Service
	// IsInitialized returns true if service registry is initialized.
	IsInitialized() bool
}

// IsRegistryInitialized returns a voter that checks if the registry is initialized.
func IsRegistryInitialized(serviceRegistry ServiceRegistry) task.Voter {
	return task.VoterFn(func() bool {
		return serviceRegistry.IsInitialized()
	})
}

// NewService creates instance of a FIMP service.
func NewService(publisher ServicePublisher, specification *fimptype.Service) Service {
	return &service{
		publisher:     publisher,
		specification: specification,
	}
}

// Service is a private implementation of a FIMP service.
type service struct {
	publisher     ServicePublisher
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
	return s.publisher.PublishServiceMessage(s, message)
}
