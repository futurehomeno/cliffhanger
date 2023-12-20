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

	PublishEvent(event ServiceEvent)
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

// ShouldSkipServiceTask returns true if service tasks should be skipped because of the thing connectivity status.
func ShouldSkipServiceTask(serviceRegistry ServiceRegistry, service Service) bool {
	// We do not skip the task is service registry is not also a thing registry.
	thingRegistry, ok := serviceRegistry.(ThingRegistry)
	if !ok {
		return false
	}

	// We do not skip the task if we cannot retrieve thing by topic.
	t := thingRegistry.ThingByTopic(service.Topic())
	if t == nil {
		return false
	}

	return t.ConnectivityReport().ConnectionStatus == ConnectionStatusDown
}

// SpecificationOption is an interface representing a particular service specification option.
type SpecificationOption interface {
	// Apply applies the option to the provided service specification.
	Apply(*fimptype.Service)
}

// SpecificationOptionFn is an convenience adapter for the SpecificationOption interface.
type SpecificationOptionFn func(*fimptype.Service)

// Apply applies the option to the provided service specification.
func (f SpecificationOptionFn) Apply(s *fimptype.Service) {
	f(s)
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

// PublishEvent publishes a service to the event manager.
func (s *service) PublishEvent(event ServiceEvent) {
	s.publisher.PublishServiceEvent(s, event)
}
