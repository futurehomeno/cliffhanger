package fanctrl

import (
	"fmt"
	"slices"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

const (
	// PropertySupportedModes is a property key for supported modes.
	PropertySupportedModes = "sup_modes"
)

// DefaultReportingStrategy is the default reporting strategy used by the service for periodic reports of state changes.
var DefaultReportingStrategy = cache.ReportOnChangeOnly()

// Controller is an interface representing an actual device using fan service.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Controller interface {
	// SetFanCtrlMode sets the mode of the device.
	SetFanCtrlMode(mode string) error
	// FanCtrlModeReport returns a current mode of the device.
	FanCtrlModeReport() (string, error)
}

// Service is an interface representing a fanctrl FIMP service.
type Service interface {
	adapter.Service

	// SetMode sets the mode of the device.
	SetMode(mode string) error
	// SendModeReport sends a current mode report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendModeReport(force bool) (bool, error)
	// SupportedModes returns a list of supported modes components.
	SupportedModes() []string
}

// Config represents a service configuration.
type Config struct {
	Specification     *fimptype.Service
	Controller        Controller
	ReportingStrategy cache.ReportingStrategy
}

// NewService creates a new instance of a fanctrl FIMP service.
func NewService(
	publisher adapter.ServicePublisher,
	cfg *Config,
) Service {
	cfg.Specification.Name = FanCtrl

	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	if cfg.ReportingStrategy == nil {
		cfg.ReportingStrategy = DefaultReportingStrategy
	}

	s := &service{
		Service:           adapter.NewService(publisher, cfg.Specification),
		controller:        cfg.Controller,
		lock:              &sync.Mutex{},
		reportingCache:    cache.NewReportingCache(),
		reportingStrategy: cfg.ReportingStrategy,
	}

	return s
}

// service is a private implementation of a fanctrl FIMP service.
type service struct {
	adapter.Service

	controller        Controller
	lock              *sync.Mutex
	reportingCache    cache.ReportingCache
	reportingStrategy cache.ReportingStrategy
}

// SetMode sets the mode of the device.
func (s *service) SetMode(mode string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !slices.Contains(s.SupportedModes(), mode) {
		return fmt.Errorf("mode %s is not supported", mode)
	}

	err := s.controller.SetFanCtrlMode(mode)
	if err != nil {
		return fmt.Errorf("failed to set mode: %w", err)
	}

	return nil
}

// SendModeReport sends a current mode report. Returns true if a report was sent.
func (s *service) SendModeReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	mode, err := s.controller.FanCtrlModeReport()
	if err != nil {
		return false, fmt.Errorf("failed to get mode: %w", err)
	}

	if !slices.Contains(s.SupportedModes(), mode) {
		return false, fmt.Errorf("mode %s is not supported", mode)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtModeReport, "", mode) {
		return false, nil
	}

	message := fimpgo.NewStringMessage(
		EvtModeReport,
		s.Name(),
		mode,
		nil,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("failed to send mode report: %w", err)
	}

	s.reportingCache.Reported(EvtModeReport, "", mode)

	return true, nil
}

// SupportedModes returns a list of supported modes components.
func (s *service) SupportedModes() []string {
	return s.Service.Specification().PropertyStrings(PropertySupportedModes)
}
