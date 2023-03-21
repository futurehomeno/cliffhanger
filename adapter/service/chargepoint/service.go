package chargepoint

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

const (
	PropertySupportedStates        = "sup_states"
	PropertySupportedChargingModes = "sup_charging_modes"
	PropertyChargingMode           = "charging_mode"

	StateReadyToCharge = "ready_to_charge"
	StateCharging      = "charging"
)

var (
	// DefaultStateReportingStrategy is the default reporting strategy used by the service for periodic reports of state changes.
	DefaultStateReportingStrategy = cache.ReportOnChangeOnly()
	// DefaultSessionReportingStrategy is the default reporting strategy used by the service for periodic reports of session.
	DefaultSessionReportingStrategy = cache.ReportAtLeastEvery(30 * time.Minute)
)

// Controller is an interface representing an actual car charger device.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Controller interface {
	// StartChargepointCharging starts car charging.
	StartChargepointCharging(mode string) error
	// StopChargepointCharging stops car charging.
	StopChargepointCharging() error
	// SetChargepointCableLock locks and unlocks the cable connector.
	SetChargepointCableLock(bool) error
	// ChargepointCableLockReport returns a current state of the chargepoint cable lock.
	ChargepointCableLockReport() (bool, error)
	// ChargepointCurrentSessionReport returns cumulative energy charged during the current session.
	ChargepointCurrentSessionReport() (float64, error)
	// ChargepointStateReport returns a current state of the chargepoint.
	ChargepointStateReport() (string, error)
}

// Service is an interface representing a waterHeater FIMP service.
type Service interface {
	adapter.Service

	// StartCharging starts car charging.
	StartCharging(mode string) error
	// StopCharging stops car charging.
	StopCharging() error
	// SetCableLock locks and unlocks the cable connector.
	SetCableLock(bool) error
	// SendCurrentSessionReport sends a current charging session report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendCurrentSessionReport(force bool) (bool, error)
	// SendCableLockReport sends a cable lock report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendCableLockReport(force bool) (bool, error)
	// SendStateReport sends a state report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendStateReport(force bool) (bool, error)
	// SupportedStates returns states that are supported by the chargepoint.
	SupportedStates() []string
}

// Config represents a service configuration.
type Config struct {
	Specification            *fimptype.Service
	Controller               Controller
	StateReportingStrategy   cache.ReportingStrategy
	SessionReportingStrategy cache.ReportingStrategy
}

// NewService creates new instance of a water heater FIMP service.
func NewService(
	a adapter.Adapter,
	cfg *Config,
) Service {
	cfg.Specification.Name = Chargepoint

	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	if cfg.SessionReportingStrategy == nil {
		cfg.SessionReportingStrategy = DefaultSessionReportingStrategy
	}

	if cfg.StateReportingStrategy == nil {
		cfg.StateReportingStrategy = DefaultStateReportingStrategy
	}

	s := &service{
		Service:                  adapter.NewService(a, cfg.Specification),
		controller:               cfg.Controller,
		lock:                     &sync.Mutex{},
		reportingCache:           cache.NewReportingCache(),
		sessionReportingStrategy: cfg.SessionReportingStrategy,
		stateReportingStrategy:   cfg.StateReportingStrategy,
	}

	return s
}

// service is a private implementation of a water heater FIMP service.
type service struct {
	adapter.Service

	controller               Controller
	lock                     *sync.Mutex
	reportingCache           cache.ReportingCache
	stateReportingStrategy   cache.ReportingStrategy
	sessionReportingStrategy cache.ReportingStrategy
}

// StartCharging starts car charging.
func (s *service) StartCharging(mode string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	mode, err := s.normalizeChargingMode(mode)
	if err != nil {
		return fmt.Errorf("%s: failed to start charging: %w", s.Name(), err)
	}

	err = s.controller.StartChargepointCharging(mode)
	if err != nil {
		return fmt.Errorf("%s: failed to start charging: %w", s.Name(), err)
	}

	return nil
}

// StopCharging stops car charging.
func (s *service) StopCharging() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := s.controller.StopChargepointCharging()
	if err != nil {
		return fmt.Errorf("%s: failed to stop charging: %w", s.Name(), err)
	}

	return nil
}

// SetCableLock locks and unlocks the cable connector.
func (s *service) SetCableLock(lock bool) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := s.controller.SetChargepointCableLock(lock)
	if err != nil {
		return fmt.Errorf("%s: failed to set cable lock to %t: %w", s.Name(), lock, err)
	}

	return nil
}

// SendCurrentSessionReport sends a current charging session report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendCurrentSessionReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	value, err := s.controller.ChargepointCurrentSessionReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve current session report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.sessionReportingStrategy, EvtCurrentSessionReport, "", value) {
		return false, nil
	}

	message := fimpgo.NewFloatMessage(
		EvtCurrentSessionReport,
		s.Name(),
		value,
		nil,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send current session report: %w", s.Name(), err)
	}

	s.reportingCache.Reported(EvtCurrentSessionReport, "", value)

	return true, nil
}

// SendCableLockReport sends a cable lock report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendCableLockReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	value, err := s.controller.ChargepointCableLockReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve cable lock report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.stateReportingStrategy, EvtCableLockReport, "", value) {
		return false, nil
	}

	message := fimpgo.NewBoolMessage(
		EvtCableLockReport,
		s.Name(),
		value,
		nil,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send cable lock report: %w", s.Name(), err)
	}

	s.reportingCache.Reported(EvtCableLockReport, "", value)

	return true, nil
}

// SendStateReport sends a state report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendStateReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	value, err := s.controller.ChargepointStateReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve state report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.stateReportingStrategy, EvtStateReport, "", value) {
		return false, nil
	}

	message := fimpgo.NewStringMessage(
		EvtStateReport,
		s.Name(),
		value,
		nil,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send state report: %w", s.Name(), err)
	}

	s.reportingCache.Reported(EvtStateReport, "", value)

	return true, nil
}

// SupportedStates returns states that are supported by the chargepoint.
func (s *service) SupportedStates() []string {
	return s.Service.Specification().PropertyStrings(PropertySupportedStates)
}

// normalizeChargingMode normalizes provided charging mode. Returns true, when everything is fine.
// TODO: consider introducing mode capable controller (which will contain start function with mode,
//
//	where the classic one will not take any parameters) and allow service to decide, which one should be used.
//
//nolint:godox
func (s *service) normalizeChargingMode(mode string) (string, error) {
	if mode == "" {
		return "", nil
	}

	supportedModes := s.Specification().PropertyStrings(PropertySupportedChargingModes)
	if len(supportedModes) == 0 {
		return "", nil
	}

	m := strings.ToLower(mode)

	for _, supported := range supportedModes {
		if m == supported {
			return m, nil
		}
	}

	return "", fmt.Errorf("unsupported mode: %s: supported modes: %v", m, supportedModes)
}
