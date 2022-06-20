package presence

import (
	"fmt"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

// DefaultReportingStrategy is the default strategy used by the service for periodic reports.
var DefaultReportingStrategy = cache.ReportOnChangeOnly()

// Controller is an interface representing an actual device.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Controller interface {
	// PresencePresenceReport returns a current presence value.
	PresencePresenceReport() (bool, error)
}

// Service is an interface representing a presence FIMP service.
type Service interface {
	adapter.Service

	// SendPresenceReport sends a presence report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendPresenceReport(force bool) (bool, error)
}

// Config represents a service configuration.
type Config struct {
	Specification     *fimptype.Service
	Controller        Controller
	ReportingStrategy cache.ReportingStrategy
}

// NewService creates a new instance of a presence FIMP service.
func NewService(
	mqtt *fimpgo.MqttTransport,
	cfg *Config,
) Service {
	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	if cfg.ReportingStrategy == nil {
		cfg.ReportingStrategy = DefaultReportingStrategy
	}

	return &service{
		Service:           adapter.NewService(mqtt, cfg.Specification),
		controller:        cfg.Controller,
		lock:              &sync.Mutex{},
		reportingStrategy: cfg.ReportingStrategy,
		reportingCache:    cache.NewReportingCache(),
	}
}

// service is a private implementation of a presence FIMP service.
type service struct {
	adapter.Service

	controller        Controller
	lock              *sync.Mutex
	reportingCache    cache.ReportingCache
	reportingStrategy cache.ReportingStrategy
}

// SendPresenceReport sends a presence report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendPresenceReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	value, err := s.controller.PresencePresenceReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to get presence reeport: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtPresenceReport, "", value) {
		return false, nil
	}

	message := fimpgo.NewBoolMessage(
		EvtPresenceReport,
		s.Name(),
		value,
		nil,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send presence report: %w", s.Name(), err)
	}

	s.reportingCache.Reported(EvtPresenceReport, "", value)

	return true, nil
}
