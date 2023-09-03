package devsys

import (
	"fmt"
	"sync"

	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
)

// Controller for dev_sys service specifies no methods, as the service has no mandatory interfaces.
type Controller interface{}

// RebootController is a controller that allows to reboot the device.
type RebootController interface {
	RebootDevice(hard bool) error
}

// Service is an interface representing a dev_sys FIMP service.
type Service interface {
	adapter.Service

	// Reboot triggers device reboot.
	Reboot(hard bool) error
	// SupportsReboot returns true if the service supports rebooting the device.
	SupportsReboot() bool
}

// Config represents a service configuration.
type Config struct {
	Specification *fimptype.Service
	Controller    Controller
}

// NewService creates a new instance of a dev_sys FIMP service.
func NewService(
	publisher adapter.ServicePublisher,
	cfg *Config,
) Service {
	cfg.Specification.Name = DevSys

	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	s := &service{
		Service:    adapter.NewService(publisher, cfg.Specification),
		controller: cfg.Controller,
		lock:       &sync.Mutex{},
	}

	if s.SupportsReboot() {
		cfg.Specification.EnsureInterfaces(rebootInterfaces()...)
	}

	return s
}

// service is a private implementation of a dev_sys FIMP service.
type service struct {
	adapter.Service

	controller Controller
	lock       *sync.Mutex
}

// Reboot triggers device reboot.
func (s *service) Reboot(hard bool) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	controller, err := s.rebootController()
	if err != nil {
		return err
	}

	err = controller.RebootDevice(hard)
	if err != nil {
		return fmt.Errorf("%s: failed to reboot device: %w", s.Name(), err)
	}

	return nil
}

// SupportsReboot returns true if the service supports rebooting the device.
func (s *service) SupportsReboot() bool {
	_, err := s.rebootController()

	return err == nil
}

// rebootController returns a reboot controller, if the service supports rebooting the device.
func (s *service) rebootController() (RebootController, error) {
	controller, ok := s.controller.(RebootController)
	if !ok {
		return nil, fmt.Errorf("%s: reboot functionality is not supported", s.Name())
	}

	return controller, nil
}
