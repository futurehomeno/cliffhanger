package parameters

import (
	"fmt"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
)

const (
	PropertyParameterSizes = "req_param_sizes"
)

// Controller is an interface representing an actual car charger device.
type Controller interface {
	SetParameter(p Parameter) error
	GetParameter(id string) (Parameter, error)
	GetSupportedParameters() ([]SupportedParameter, error) // TODO: there should be an interface composition implemented here
}

// Service is an interface representing a waterHeater FIMP service.
type Service interface {
	adapter.Service

	SetParameter(p Parameter) error
	SendParameterReport(id string) error
	SendSupportedParamsReport() error
	SupportsParamsDiscovery() bool
}

// Config represents a service configuration.
type Config struct {
	Specification *fimptype.Service
	Controller    Controller
}

// NewService creates new instance of a water heater FIMP service.
func NewService(
	publisher adapter.ServicePublisher,
	cfg *Config,
) Service {
	cfg.Specification.Name = Parameters
	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	s := &service{
		Service:    adapter.NewService(publisher, cfg.Specification),
		controller: cfg.Controller,
		lock:       &sync.Mutex{},
	}

	if s.SupportsParamsDiscovery() {
		s.Specification().EnsureInterfaces(optionalInterfaces()...)
	}

	return s
}

// service is a private implementation of a water heater FIMP service.
type service struct {
	adapter.Service

	lock       *sync.Mutex
	controller Controller
}

func (s *service) SetParameter(p Parameter) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if err := p.Validate(); err != nil {
		return fmt.Errorf("%s: invalid parameter: %w", s.Name(), err)
	}

	if err := s.controller.SetParameter(p); err != nil {
		return fmt.Errorf("%s: failed to set parameter ID %s: %w", s.Name(), p.ID, err)
	}

	return nil
}

func (s *service) SendParameterReport(id string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	parameter, err := s.controller.GetParameter(id)
	if err != nil {
		return fmt.Errorf("%s: failed to get parameter ID %s: %w", s.Name(), id, err)
	}

	if err = parameter.Validate(); err != nil {
		return fmt.Errorf("%s: invalid parameter provided: %w", s.Name(), err)
	}

	message := fimpgo.NewObjectMessage(
		EvtParamReport,
		s.Name(),
		parameter,
		nil,
		nil,
		nil,
	)
	message.WithStorageStrategy(fimpgo.StorageStrategyAggregate, parameter.ID)

	if err = s.SendMessage(message); err != nil {
		return fmt.Errorf("%s: failed to send parameter report: %w", s.Name(), err)
	}

	return nil
}

func (s *service) SendSupportedParamsReport() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	parameters, err := s.controller.GetSupportedParameters()
	if err != nil {
		return fmt.Errorf("%s: failed to get supported parameters: %w", s.Name(), err)
	}

	for _, p := range parameters {
		if err = p.Validate(); err != nil {
			return fmt.Errorf("%s: invalid parameter provided: %w", s.Name(), err)
		}
	}

	message := fimpgo.NewObjectMessage(
		EvtSupParamsReport,
		s.Name(),
		parameters,
		nil,
		nil,
		nil,
	)

	if err = s.SendMessage(message); err != nil {
		return fmt.Errorf("%s: failed to send supported parameters report: %w", s.Name(), err)
	}

	return nil
}

func (s *service) SupportsParamsDiscovery() bool {
	sizes := s.Specification().PropertyIntegers(PropertyParameterSizes)

	return len(sizes) == 0
}
