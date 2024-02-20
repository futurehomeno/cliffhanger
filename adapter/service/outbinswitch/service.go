package outbinswitch

import (
	"fmt"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

// DefaultReportingStrategy is the default reporting strategy used by the service for periodic reports.
var DefaultReportingStrategy = cache.ReportOnChangeOnly()

// Controller is an interface representing an actual device.
type Controller interface {
	// BinarySwitchStateReport returns a current binary switch state.
	BinarySwitchStateReport() (bool, error)
	// SetBinarySwitchState sets a binary switch state.
	SetBinarySwitchState(bool) error
}

// Service is an interface representing a outbinswitch FIMP service.
type Service interface {
	adapter.Service

	SendBinaryReport(force bool) (bool, error)
	SetBinaryState(state bool) error
}

// Config represents a service configuration.
type Config struct {
	Specification     *fimptype.Service
	Controller        Controller
	ReportingStrategy cache.ReportingStrategy
}

// NewService creates a new instance of a output binary switch FIMP service.
func NewService(
	publisher adapter.ServicePublisher,
	cfg *Config,
) Service {
	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	if cfg.ReportingStrategy == nil {
		cfg.ReportingStrategy = DefaultReportingStrategy
	}

	return &service{
		Service:           adapter.NewService(publisher, cfg.Specification),
		controller:        cfg.Controller,
		reportingStrategy: cfg.ReportingStrategy,

		reportingCache: cache.NewReportingCache(),
		lock:           &sync.Mutex{},
	}
}

// service is a private implementation of a output binary switch FIMP service.
type service struct {
	adapter.Service

	controller        Controller
	lock              *sync.Mutex
	reportingCache    cache.ReportingCache
	reportingStrategy cache.ReportingStrategy
}

// SendBinaryReport sends a binary report. Returns true if a report was sent.
func (s *service) SendBinaryReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	value, err := s.controller.BinarySwitchStateReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to get binary report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtBinaryReport, "", value) {
		return false, nil
	}

	message := fimpgo.NewBoolMessage(EvtBinaryReport, s.Name(), value, nil, nil, nil)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send binary report: %w", s.Name(), err)
	}

	s.reportingCache.Reported(EvtBinaryReport, "", value)

	return true, nil
}

// SetBinaryState sets a binary state.
func (s *service) SetBinaryState(state bool) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := s.controller.SetBinarySwitchState(state)
	if err != nil {
		return fmt.Errorf("%s: failed to set binary state: %w", s.Name(), err)
	}

	return nil
}
