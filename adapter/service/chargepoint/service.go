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
	StartChargepointCharging(settings *ChargingSettings) error
	// StopChargepointCharging stops car charging.
	StopChargepointCharging() error
	// SetChargepointCableLock locks and unlocks the cable connector.
	SetChargepointCableLock(bool) error
	// ChargepointCableLockReport returns a current state of the chargepoint cable lock.
	ChargepointCableLockReport() (*CableReport, error)
	// ChargepointCurrentSessionReport returns cumulative energy charged during the current session.
	ChargepointCurrentSessionReport() (*SessionReport, error)
	// ChargepointStateReport returns a current state of the chargepoint.
	ChargepointStateReport() (State, error)
}

// AdjustableMaxCurrentController is an interface representing capability of a charger device to adjust charging current.
type AdjustableMaxCurrentController interface {
	// SetChargepointMaxCurrent sets max current of a chargepoint.
	SetChargepointMaxCurrent(int64) error
	// ChargepointMaxCurrentReport returns max current of a chargepoint.
	ChargepointMaxCurrentReport() (int64, error)
}

type AdjustableOfferedCurrentController interface {
	// SetChargepointOfferedCurrent sets offered current of a current session.
	SetChargepointOfferedCurrent(int64) error
}

// AdjustablePhaseModeController is an interface representing capability of a charger device to adjust phase mode.
type AdjustablePhaseModeController interface {
	// SetChargepointPhaseMode sets phase mode of a chargepoint.
	SetChargepointPhaseMode(PhaseMode) error
	// ChargepointPhaseModeReport returns phase mode of a chargepoint.
	ChargepointPhaseModeReport() (PhaseMode, error)
}

// Service is an interface representing a waterHeater FIMP service.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true for all send methods.
type Service interface {
	adapter.Service

	// StartCharging starts car charging.
	StartCharging(settings *ChargingSettings) error
	// StopCharging stops car charging.
	StopCharging() error
	// SetCableLock locks and unlocks the cable connector.
	SetCableLock(bool) error
	// SetOfferedCurrent sets offered current of a current session.
	SetOfferedCurrent(int64) error
	// SetMaxCurrent sets max current of a chargepoint.
	SetMaxCurrent(int64) error
	// SetPhaseMode sets phase mode of a chargepoint.
	SetPhaseMode(PhaseMode) error
	// SendCurrentSessionReport sends a current charging session report. Returns true if a report was sent.
	SendCurrentSessionReport(force bool) (bool, error)
	// SendCableLockReport sends a cable lock report. Returns true if a report was sent.
	SendCableLockReport(force bool) (bool, error)
	// SendStateReport sends a state report. Returns true if a report was sent.
	SendStateReport(force bool) (bool, error)
	// SendMaxCurrentReport sends a max current report. Returns true if a report was sent.
	SendMaxCurrentReport(force bool) (bool, error)
	// SendPhaseModeReport sends a phase mode report. Returns true if a report was sent.
	SendPhaseModeReport(force bool) (bool, error)
	// SupportedStates returns states that are supported by the chargepoint.
	SupportedStates() []string
	// SupportsAdjustingMaxCurrent returns true if the chargepoint supports adjusting current.
	SupportsAdjustingMaxCurrent() bool
	// SupportsAdjustingPhaseModes returns true if the chargepoint supports adjusting phase modes.
	SupportsAdjustingPhaseModes() bool
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
	publisher adapter.ServicePublisher,
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
		Service:                  adapter.NewService(publisher, cfg.Specification),
		controller:               cfg.Controller,
		lock:                     &sync.Mutex{},
		reportingCache:           cache.NewReportingCache(),
		sessionReportingStrategy: cfg.SessionReportingStrategy,
		stateReportingStrategy:   cfg.StateReportingStrategy,
	}

	if s.SupportsAdjustingMaxCurrent() {
		cfg.Specification.EnsureInterfaces(adjustableMaxCurrentInterfaces()...)
	}

	if s.SupportsAdjustingOfferedCurrent() {
		cfg.Specification.EnsureInterfaces(adjustableOfferedCurrentInterfaces()...)
	}

	if s.SupportsAdjustingPhaseModes() {
		cfg.Specification.EnsureInterfaces(adjustablePhaseModeInterfaces()...)
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
func (s *service) StartCharging(settings *ChargingSettings) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	var err error

	settings.Mode, err = s.normalizeChargingMode(settings.Mode)
	if err != nil {
		return fmt.Errorf("%s: failed to start charging: %w", s.Name(), err)
	}

	err = s.controller.StartChargepointCharging(settings)
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

// SetOfferedCurrent sets offered current of a current session.
func (s *service) SetOfferedCurrent(current int64) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	controller, err := s.adjustableOfferedCurrentController()
	if err != nil {
		return err
	}

	err = s.validateCurrent(current)
	if err != nil {
		return err
	}

	err = controller.SetChargepointOfferedCurrent(current)
	if err != nil {
		return fmt.Errorf("%s: failed to set offered current to %d: %w", s.Name(), current, err)
	}

	return nil
}

// SetMaxCurrent sets max current of a chargepoint.
func (s *service) SetMaxCurrent(current int64) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	controller, err := s.adjustableMaxCurrentController()
	if err != nil {
		return err
	}

	err = s.validateCurrent(current)
	if err != nil {
		return err
	}

	err = controller.SetChargepointMaxCurrent(current)
	if err != nil {
		return fmt.Errorf("%s: failed to set max current to %d: %w", s.Name(), current, err)
	}

	return nil
}

// SetPhaseMode sets phase mode of a chargepoint.
func (s *service) SetPhaseMode(mode PhaseMode) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	controller, err := s.adjustablePhaseModeController()
	if err != nil {
		return err
	}

	err = s.validatePhaseMode(mode)
	if err != nil {
		return err
	}

	err = controller.SetChargepointPhaseMode(mode)
	if err != nil {
		return fmt.Errorf("%s: failed to set phase mode to %s: %w", s.Name(), mode.String(), err)
	}

	return nil
}

// SendCurrentSessionReport sends a current charging session report. Returns true if a report was sent.
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
		value.SessionEnergy,
		value.reportProperties(s.SupportsAdjustingMaxCurrent()),
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
		value.CableLock,
		value.reportProperties(),
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
		value.String(),
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

// SendMaxCurrentReport sends a max current report. Returns true if a report was sent.
func (s *service) SendMaxCurrentReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	controller, err := s.adjustableMaxCurrentController()
	if err != nil {
		return false, err
	}

	value, err := controller.ChargepointMaxCurrentReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve max current report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.stateReportingStrategy, EvtMaxCurrentReport, "", value) {
		return false, nil
	}

	message := fimpgo.NewIntMessage(
		EvtMaxCurrentReport,
		s.Name(),
		value,
		nil,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send max current report: %w", s.Name(), err)
	}

	s.reportingCache.Reported(EvtMaxCurrentReport, "", value)

	return true, nil
}

// SendPhaseModeReport sends a phase mode report. Returns true if a report was sent.
func (s *service) SendPhaseModeReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	controller, err := s.adjustablePhaseModeController()
	if err != nil {
		return false, err
	}

	value, err := controller.ChargepointPhaseModeReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve phase mode report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.stateReportingStrategy, EvtPhaseModeReport, "", value) {
		return false, nil
	}

	message := fimpgo.NewStringMessage(
		EvtPhaseModeReport,
		s.Name(),
		value.String(),
		nil,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send phase mode report: %w", s.Name(), err)
	}

	s.reportingCache.Reported(EvtPhaseModeReport, "", value)

	return true, nil
}

// SupportedStates returns states that are supported by the chargepoint.
func (s *service) SupportedStates() []string {
	return s.Service.Specification().PropertyStrings(PropertySupportedStates)
}

// SupportsAdjustingMaxCurrent returns true if the chargepoint supports adjusting current.
func (s *service) SupportsAdjustingMaxCurrent() bool {
	_, err := s.adjustableMaxCurrentController()

	return err == nil
}

// SupportsAdjustingOfferedCurrent returns true if the chargepoint supports adjusting current.
func (s *service) SupportsAdjustingOfferedCurrent() bool {
	_, err := s.adjustableOfferedCurrentController()

	return err == nil
}

// SupportsAdjustingPhaseModes returns true if the chargepoint supports adjusting phase modes.
func (s *service) SupportsAdjustingPhaseModes() bool {
	_, err := s.adjustablePhaseModeController()

	return err == nil
}

// adjustableMaxCurrentController returns the AdjustableMaxCurrentController, if supported.
func (s *service) adjustableMaxCurrentController() (AdjustableMaxCurrentController, error) {
	_, ok := s.Specification().PropertyInteger(PropertySupportedMaxCurrent)
	if !ok {
		return nil, fmt.Errorf("%s: adjusting max current is not supported", s.Name())
	}

	controller, ok := s.controller.(AdjustableMaxCurrentController)
	if !ok {
		return nil, fmt.Errorf("%s: adjusting max current is not supported", s.Name())
	}

	return controller, nil
}

// adjustableOfferedCurrentController returns the AdjustableOfferedCurrentController, if supported.
func (s *service) adjustableOfferedCurrentController() (AdjustableOfferedCurrentController, error) {
	_, ok := s.Specification().PropertyInteger(PropertyOfferedCurrent)
	if !ok {
		return nil, fmt.Errorf("%s: adjusting offered current is not supported", s.Name())
	}

	controller, ok := s.controller.(AdjustableOfferedCurrentController)
	if !ok {
		return nil, fmt.Errorf("%s: adjusting offered current is not supported", s.Name())
	}

	return controller, nil
}

// adjustablePhaseModeController returns the AdjustablePhaseModeController, if supported.
func (s *service) adjustablePhaseModeController() (AdjustablePhaseModeController, error) {
	phaseModes := s.Specification().PropertyStrings(PropertySupportedPhaseModes)
	if len(phaseModes) == 0 {
		return nil, fmt.Errorf("%s: adjusting phase modes is not supported", s.Name())
	}

	controller, ok := s.controller.(AdjustablePhaseModeController)
	if !ok {
		return nil, fmt.Errorf("%s: adjusting phase modes is not supported", s.Name())
	}

	return controller, nil
}

// normalizeChargingMode normalizes provided charging mode, if supported.
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

// validateCurrent validates provided current.
func (s *service) validateCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("%s: configured current must be at least 6A, received %dA instead", s.Name(), current)
	}

	maximumCurrent, _ := s.Specification().PropertyInteger(PropertySupportedMaxCurrent)

	if current > maximumCurrent {
		return fmt.Errorf("%s: configured current must not exceed %dA, received %dA instead", s.Name(), maximumCurrent, current)
	}

	return nil
}

// validatePhaseMode validates provided phase mode.
func (s *service) validatePhaseMode(mode PhaseMode) error {
	supportedModes := s.Specification().PropertyStrings(PropertySupportedPhaseModes)

	for _, supportedMode := range supportedModes {
		if mode.String() == supportedMode {
			return nil
		}
	}

	return fmt.Errorf("%s: unsupported phase mode: %s", s.Name(), mode.String())
}
