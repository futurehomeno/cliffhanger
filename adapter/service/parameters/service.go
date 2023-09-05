package parameters

import (
	"fmt"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

// Controller is an interface representing a device holding parameters.
type Controller interface {
	// SetParameter sets a parameter.
	SetParameter(p *Parameter) error
	// GetParameter returns a parameter by ID.
	GetParameter(id string) (*Parameter, error)
	// GetParameterSpecifications returns a list of all parameter specifications/definitions.
	GetParameterSpecifications() ([]*ParameterSpecification, error)
}

// Service is an interface representing a parameters FIMP service.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true for all send methods.
type Service interface {
	adapter.Service

	// SetParameter sets a parameter.
	SetParameter(p *Parameter) error
	// SendParameterReport sends a parameter report. Returns true if the report was sent.
	SendParameterReport(id string, force bool) (bool, error)
	// SendSupportedParamsReport sends a supported parameters report. Returns true if the report was sent.
	SendSupportedParamsReport(force bool) (bool, error)
}

// Config represents a service configuration.
type Config struct {
	Specification *fimptype.Service
	Controller    Controller
}

// NewService creates new instance of a parameters FIMP service.
func NewService(
	publisher adapter.ServicePublisher,
	cfg *Config,
) Service {
	cfg.Specification.Name = Parameters
	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	return &service{
		Service:           adapter.NewService(publisher, cfg.Specification),
		controller:        cfg.Controller,
		lock:              &sync.Mutex{},
		reportingCache:    cache.NewReportingCache(),
		reportingStrategy: cache.ReportOnChangeOnly(),
	}
}

// service is a private implementation of a parameters FIMP service.
type service struct {
	adapter.Service

	lock              *sync.Mutex
	controller        Controller
	reportingCache    cache.ReportingCache
	reportingStrategy cache.ReportingStrategy
}

func (s *service) SetParameter(p *Parameter) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if err := p.Validate(); err != nil {
		return fmt.Errorf("%s: failed to Validate parameter: %w", s.Name(), err)
	}

	spec, err := s.getParameterSpecification(p.ID)
	if err != nil {
		return fmt.Errorf("%s: failed to get parameter specification: %w", s.Name(), err)
	}

	if spec.ReadOnly {
		return fmt.Errorf("%s: parameter is read only", s.Name())
	}

	if err = spec.ValidateParameter(p); err != nil {
		return fmt.Errorf("%s: failed to Validate parameter: %w", s.Name(), err)
	}

	if err = s.controller.SetParameter(p); err != nil {
		return fmt.Errorf("%s: failed to set parameter ID %s: %w", s.Name(), p.ID, err)
	}

	return nil
}

func (s *service) SendParameterReport(id string, force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	parameter, err := s.controller.GetParameter(id)
	if err != nil {
		return false, fmt.Errorf("%s: failed to get parameter ID %s: %w", s.Name(), id, err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtParamReport, "", parameter) {
		return false, nil
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
		return false, fmt.Errorf("%s: failed to send parameter report: %w", s.Name(), err)
	}

	return true, nil
}

func (s *service) SendSupportedParamsReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	parameters, err := s.controller.GetParameterSpecifications()
	if err != nil {
		return false, fmt.Errorf("%s: failed to get supported parameters: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtSupParamsReport, "", parameters) {
		return false, nil
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
		return false, fmt.Errorf("%s: failed to send supported parameters report: %w", s.Name(), err)
	}

	return true, nil
}

func (s *service) getParameterSpecification(id string) (*ParameterSpecification, error) {
	specs, err := s.controller.GetParameterSpecifications()
	if err != nil {
		return nil, fmt.Errorf("failed to get parameter specifications: %w", err)
	}

	for _, spec := range specs {
		if spec.ID == id {
			return spec, nil
		}
	}

	return nil, fmt.Errorf("parameter specification id '%s' not found", id)
}
