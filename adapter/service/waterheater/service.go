package waterheater

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

// Constants defining important properties specific for the service.
const (
	UnitC = "C"
	UnitF = "F"

	ModeOff = "off"

	StateHeat = "heat"
	StateIdle = "idle"

	PropertySupportedModes     = "sup_modes"
	PropertySupportedSetpoints = "sup_setpoints"
	PropertySupportedStates    = "sup_states"
	PropertySupportedRange     = "sup_range"
	PropertySupportedRanges    = "sup_ranges"
	PropertySupportedStep      = "sup_step"
)

// DefaultReportingStrategy is the default reporting strategy used by the service for periodic reports.
var DefaultReportingStrategy = cache.ReportOnChangeOnly()

// Controller is an interface representing an actual water heating device.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Controller interface {
	// SetWaterHeaterMode sets a new waterHeater mode.
	SetWaterHeaterMode(mode string) error
	// SetWaterHeaterSetpoint sets a setpoint for a particular mode.
	SetWaterHeaterSetpoint(mode string, value float64, unit string) error
	// WaterHeaterModeReport returns a current mode information.
	WaterHeaterModeReport() (mode string, err error)
	// WaterHeaterSetpointReport returns a current setpoint for given mode.
	WaterHeaterSetpointReport(mode string) (value float64, unit string, err error)
	// WaterHeaterStateReport returns a current state of the water heater.
	WaterHeaterStateReport() (string, error)
}

// Service is an interface representing a water heater FIMP service.
type Service interface {
	adapter.Service

	// SetMode sets the mode of the device.
	SetMode(mode string) error
	// SetSetpoint sets the setpoint for a specific mode. Unit value is ignored and maintained for informational purpose only.
	SetSetpoint(mode string, value float64, unit string) error
	// SendModeReport sends a mode report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendModeReport(force bool) (bool, error)
	// SendSetpointReport sends a setpoint report based on provided mode. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendSetpointReport(mode string, force bool) (bool, error)
	// SendStateReport sends a state report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendStateReport(force bool) (bool, error)
	// SupportedModes returns modes that are supported by the waterHeater.
	SupportedModes() []string
	// SupportedSetpoints returns setpoints that are supported by the waterHeater.
	SupportedSetpoints() []string
	// SupportedStates returns states that are supported by the waterHeater.
	SupportedStates() []string
	// SupportsSetpoint returns true if provided setpoint mode is supported.
	SupportsSetpoint(setpoint string) bool
}

// Config represents a service configuration.
type Config struct {
	Specification     *fimptype.Service
	Controller        Controller
	ReportingStrategy cache.ReportingStrategy
}

// NewService creates new instance of a water heater FIMP service.
func NewService(
	publisher adapter.ServicePublisher,
	cfg *Config,
) Service {
	cfg.Specification.Name = WaterHeater

	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	if cfg.ReportingStrategy == nil {
		cfg.ReportingStrategy = DefaultReportingStrategy
	}

	return &service{
		Service:           adapter.NewService(publisher, cfg.Specification),
		controller:        cfg.Controller,
		lock:              &sync.Mutex{},
		reportingStrategy: cfg.ReportingStrategy,
		reportingCache:    cache.NewReportingCache(),
	}
}

// service is a private implementation of a water heater FIMP service.
type service struct {
	adapter.Service

	controller        Controller
	lock              *sync.Mutex
	reportingCache    cache.ReportingCache
	reportingStrategy cache.ReportingStrategy
}

// SetMode sets mode of the device.
func (s *service) SetMode(mode string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	normalizedMode, ok := s.normalizeMode(mode)
	if !ok {
		return fmt.Errorf("%s: mode is unsupported: %s", s.Name(), mode)
	}

	err := s.controller.SetWaterHeaterMode(normalizedMode)
	if err != nil {
		return fmt.Errorf("%s: failed to set mode %s: %w", s.Name(), normalizedMode, err)
	}

	return nil
}

// SetSetpoint sets setpoint for a specific mode. Unit value is ignored and maintained for informational purpose only.
func (s *service) SetSetpoint(mode string, value float64, unit string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	normalizedMode, ok := s.normalizeSetpoint(mode)
	if !ok {
		return fmt.Errorf("%s: setpoint mode is unsupported: %s", s.Name(), mode)
	}

	normalizedValue, err := s.normalizeValue(mode, value)
	if err != nil {
		return fmt.Errorf("%s: setpoint value is incorrect: %w", s.Name(), err)
	}

	err = s.controller.SetWaterHeaterSetpoint(normalizedMode, normalizedValue, unit)
	if err != nil {
		return fmt.Errorf("%s: failed to set setpoint for mode %s for value %.01f: %w", s.Name(), normalizedMode, normalizedValue, err)
	}

	return nil
}

// SendModeReport sends a mode report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendModeReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	value, err := s.controller.WaterHeaterModeReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve mode report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtModeReport, "", value) {
		return false, nil
	}

	message := fimpgo.NewStringMessage(
		EvtModeReport,
		s.Name(),
		value,
		nil,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send mode report: %w", s.Name(), err)
	}

	s.reportingCache.Reported(EvtModeReport, "", value)

	return true, nil
}

// SendSetpointReport sends a setpoint report based on provided mode. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendSetpointReport(mode string, force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	normalizedMode, ok := s.normalizeSetpoint(mode)
	if !ok {
		return false, fmt.Errorf("%s: setpoint mode is unsupported: %s", s.Name(), mode)
	}

	value, unit, err := s.controller.WaterHeaterSetpointReport(normalizedMode)
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve setpoint report for mode %s: %w", s.Name(), normalizedMode, err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtSetpointReport, mode, value) {
		return false, nil
	}

	message := fimpgo.NewObjectMessage(
		EvtSetpointReport,
		s.Name(),
		NewSetpoint(normalizedMode, value, unit),
		nil,
		nil,
		nil,
	).WithStorageStrategy(fimpgo.StorageStrategyAggregate, normalizedMode)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send state report: %w", s.Name(), err)
	}

	s.reportingCache.Reported(EvtSetpointReport, mode, value)

	return true, nil
}

// SendStateReport sends a state report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendStateReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	value, err := s.controller.WaterHeaterStateReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve state report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtStateReport, "", value) {
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

// SupportedModes returns modes that are supported by the water heater.
func (s *service) SupportedModes() []string {
	return s.Service.Specification().PropertyStrings(PropertySupportedModes)
}

// SupportedSetpoints returns setpoints that are supported by the water heater.
func (s *service) SupportedSetpoints() []string {
	return s.Service.Specification().PropertyStrings(PropertySupportedSetpoints)
}

// SupportedStates returns states that are supported by the water heater.
func (s *service) SupportedStates() []string {
	return s.Service.Specification().PropertyStrings(PropertySupportedStates)
}

// SupportsSetpoint returns true if provided setpoint mode is supported.
func (s *service) SupportsSetpoint(setpoint string) bool {
	_, ok := s.normalizeSetpoint(setpoint)

	return ok
}

// normalizeMode checks if mode is supported and returns its normalized form.
func (s *service) normalizeMode(mode string) (string, bool) {
	for _, value := range s.SupportedModes() {
		if strings.EqualFold(mode, value) {
			return value, true
		}
	}

	return "", false
}

// normalizeSetpoint checks if setpoint is supported and returns its normalized form.
func (s *service) normalizeSetpoint(mode string) (string, bool) {
	for _, value := range s.SupportedSetpoints() {
		if strings.EqualFold(mode, value) {
			return value, true
		}
	}

	return "", false
}

// normalizeValue normalizes setpoint value for a specific mode.
func (s *service) normalizeValue(mode string, value float64) (float64, error) {
	step, ok := s.Service.Specification().PropertyFloat(PropertySupportedStep)
	if ok && step > 0 {
		value = math.Round(value/step) * step
	}

	supportedRange := s.supportedRange(mode)
	if supportedRange == nil {
		return value, nil
	}

	if value < supportedRange.Min || value > supportedRange.Max {
		return 0, fmt.Errorf("%s: value %.01f is out of range %.01f - %.01f for mode %s",
			s.Name(), value, supportedRange.Min, supportedRange.Max, mode,
		)
	}

	return value, nil
}

// supportedRange returns range supported for a given mode.
func (s *service) supportedRange(mode string) *Range {
	var supportedRanges map[string]*Range

	_ = s.Service.Specification().PropertyObject(PropertySupportedRanges, &supportedRanges)

	supportedRange, ok := supportedRanges[mode]
	if ok {
		return supportedRange
	}

	supportedRange = &Range{}
	ok = s.Service.Specification().PropertyObject(PropertySupportedRange, supportedRange)

	if ok {
		return supportedRange
	}

	return nil
}

// Setpoint is an object representing a water heater setpoint.
type Setpoint struct {
	Type        string  `json:"type"`
	Temperature float64 `json:"temp"`
	Unit        string  `json:"unit"`
}

// NewSetpoint create a new setpoint object.
func NewSetpoint(mode string, temp float64, unit string) *Setpoint {
	return &Setpoint{
		Type:        mode,
		Temperature: temp,
		Unit:        unit,
	}
}
