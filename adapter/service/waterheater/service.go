package waterheater

import (
	"fmt"
	"strings"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
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
)

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

// NewService creates new instance of a water heater FIMP service.
func NewService(
	mqtt *fimpgo.MqttTransport,
	specification *fimptype.Service,
	controller Controller,
) Service {
	specification.Name = WaterHeater

	specification.EnsureInterfaces(requiredInterfaces()...)

	return &service{
		Service:    adapter.NewService(mqtt, specification),
		controller: controller,
		lock:       &sync.Mutex{},
	}
}

// service is a private implementation of a water heater FIMP service.
type service struct {
	adapter.Service

	controller Controller
	lock       *sync.Mutex
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

	err := s.controller.SetWaterHeaterSetpoint(normalizedMode, value, unit)
	if err != nil {
		return fmt.Errorf("%s: failed to set setpoint for mode %s for value %.01f: %w", s.Name(), normalizedMode, value, err)
	}

	return nil
}

// SendModeReport sends a mode report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendModeReport(_ bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	value, err := s.controller.WaterHeaterModeReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve mode report: %w", s.Name(), err)
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

	return true, nil
}

// SendSetpointReport sends a setpoint report based on provided mode. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendSetpointReport(mode string, _ bool) (bool, error) {
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

	message := fimpgo.NewObjectMessage(
		EvtSetpointReport,
		s.Name(),
		NewSetpoint(normalizedMode, value, unit),
		nil,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send state report: %w", s.Name(), err)
	}

	return true, nil
}

// SendStateReport sends a state report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendStateReport(_ bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	value, err := s.controller.WaterHeaterStateReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve state report: %w", s.Name(), err)
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
