package diagnostic

import (
	"fmt"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

// DefaultReportingStrategy is the default state reporting strategy used by the service for periodic reports of state changes.
var DefaultReportingStrategy = cache.ReportAtLeastEvery(30 * time.Minute)

// Controller for diagnostic service specifies no methods, as the service has no mandatory interfaces.
type Controller any

// LQIReporter is an optional controller returning the current Link Quality Indicator value.
type LQIReporter interface {
	LQIReport() (int, error)
}

// RSSIReporter is an optional controller returning the current Received Signal Strength Indicator value.
type RSSIReporter interface {
	RSSIReport() (int, error)
}

// RebootReasonReporter is an optional controller returning the reason of the last device reboot.
type RebootReasonReporter interface {
	RebootReasonReport() (string, error)
}

// RebootsCountReporter is an optional controller returning the number of times the device has been rebooted.
type RebootsCountReporter interface {
	RebootsCountReport() (int, error)
}

// Service is an interface representing a diagnostic FIMP service.
type Service interface {
	adapter.Service

	// SendLQIReport sends a Link Quality Indicator report. Returns true if a report was sent.
	SendLQIReport(force bool) (bool, error)
	// SendRSSIReport sends a Received Signal Strength Indicator report. Returns true if a report was sent.
	SendRSSIReport(force bool) (bool, error)
	// SendRebootReasonReport sends a reboot reason report. Returns true if a report was sent.
	SendRebootReasonReport(force bool) (bool, error)
	// SendRebootsCountReport sends a reboots count report. Returns true if a report was sent.
	SendRebootsCountReport(force bool) (bool, error)

	// SupportsLQI returns true if the controller can report the Link Quality Indicator.
	SupportsLQI() bool
	// SupportsRSSI returns true if the controller can report the Received Signal Strength Indicator.
	SupportsRSSI() bool
	// SupportsRebootReason returns true if the controller can report the last reboot reason.
	SupportsRebootReason() bool
	// SupportsRebootsCount returns true if the controller can report the reboots count.
	SupportsRebootsCount() bool
}

// Config represents a service configuration.
type Config struct {
	Specification     *fimptype.Service
	Controller        Controller
	ReportingStrategy cache.ReportingStrategy
}

// NewService creates a new instance of a diagnostic FIMP service.
func NewService(
	publisher adapter.ServicePublisher,
	cfg *Config,
) Service {
	cfg.Specification.Name = Diagnostic

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

	if s.SupportsLQI() {
		cfg.Specification.EnsureInterfaces(lqiInterfaces()...)
	}

	if s.SupportsRSSI() {
		cfg.Specification.EnsureInterfaces(rssiInterfaces()...)
	}

	if s.SupportsRebootReason() {
		cfg.Specification.EnsureInterfaces(rebootReasonInterfaces()...)
	}

	if s.SupportsRebootsCount() {
		cfg.Specification.EnsureInterfaces(rebootsCountInterfaces()...)
	}

	return s
}

// service is a private implementation of a diagnostic FIMP service.
type service struct {
	adapter.Service

	controller        Controller
	lock              *sync.Mutex
	reportingCache    cache.ReportingCache
	reportingStrategy cache.ReportingStrategy
}

// SendLQIReport sends a Link Quality Indicator report. Returns true if a report was sent.
func (s *service) SendLQIReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	controller, ok := s.controller.(LQIReporter)
	if !ok {
		return false, fmt.Errorf("%s: LQI reporting is not supported", s.Name())
	}

	value, err := controller.LQIReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve LQI report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtLQIReport, "", value) {
		return false, nil
	}

	if err := s.SendMessage(fimpgo.NewIntMessage(EvtLQIReport, s.Name(), value, nil, nil, nil)); err != nil {
		return false, fmt.Errorf("%s: failed to send LQI report: %w", s.Name(), err)
	}

	s.reportingCache.Reported(EvtLQIReport, "", value)

	return true, nil
}

// SendRSSIReport sends a Received Signal Strength Indicator report. Returns true if a report was sent.
func (s *service) SendRSSIReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	controller, ok := s.controller.(RSSIReporter)
	if !ok {
		return false, fmt.Errorf("%s: RSSI reporting is not supported", s.Name())
	}

	value, err := controller.RSSIReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve RSSI report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtRSSIReport, "", value) {
		return false, nil
	}

	if err := s.SendMessage(fimpgo.NewIntMessage(EvtRSSIReport, s.Name(), value, nil, nil, nil)); err != nil {
		return false, fmt.Errorf("%s: failed to send RSSI report: %w", s.Name(), err)
	}

	s.reportingCache.Reported(EvtRSSIReport, "", value)

	return true, nil
}

// SendRebootReasonReport sends a reboot reason report. Returns true if a report was sent.
func (s *service) SendRebootReasonReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	controller, ok := s.controller.(RebootReasonReporter)
	if !ok {
		return false, fmt.Errorf("%s: reboot reason reporting is not supported", s.Name())
	}

	value, err := controller.RebootReasonReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve reboot reason report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtRebootReasonReport, "", value) {
		return false, nil
	}

	if err := s.SendMessage(fimpgo.NewStringMessage(EvtRebootReasonReport, s.Name(), value, nil, nil, nil)); err != nil {
		return false, fmt.Errorf("%s: failed to send reboot reason report: %w", s.Name(), err)
	}

	s.reportingCache.Reported(EvtRebootReasonReport, "", value)

	return true, nil
}

// SendRebootsCountReport sends a reboots count report. Returns true if a report was sent.
func (s *service) SendRebootsCountReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	controller, ok := s.controller.(RebootsCountReporter)
	if !ok {
		return false, fmt.Errorf("%s: reboots count reporting is not supported", s.Name())
	}

	value, err := controller.RebootsCountReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve reboots count report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtRebootCountReport, "", value) {
		return false, nil
	}

	if err := s.SendMessage(fimpgo.NewIntMessage(EvtRebootCountReport, s.Name(), value, nil, nil, nil)); err != nil {
		return false, fmt.Errorf("%s: failed to send reboots count report: %w", s.Name(), err)
	}

	s.reportingCache.Reported(EvtRebootCountReport, "", value)

	return true, nil
}

// SupportsLQI returns true if the controller can report the Link Quality Indicator.
func (s *service) SupportsLQI() bool {
	_, ok := s.controller.(LQIReporter)

	return ok
}

// SupportsRSSI returns true if the controller can report the Received Signal Strength Indicator.
func (s *service) SupportsRSSI() bool {
	_, ok := s.controller.(RSSIReporter)

	return ok
}

// SupportsRebootReason returns true if the controller can report the last reboot reason.
func (s *service) SupportsRebootReason() bool {
	_, ok := s.controller.(RebootReasonReporter)

	return ok
}

// SupportsRebootsCount returns true if the controller can report the reboots count.
func (s *service) SupportsRebootsCount() bool {
	_, ok := s.controller.(RebootsCountReporter)

	return ok
}
