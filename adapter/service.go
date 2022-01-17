package adapter

import (
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
}

// NewService creates instance of a FIMP service.
func NewService(specification *fimptype.Service) Service {
	return &service{
		specification: specification,
	}
}

// Service is a private implementation of a FIMP service.
type service struct {
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
