package thermostat

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
)

// Constants defining important properties specific for the service
const (
	UnitC = "C"
	UnitF = "F"

	ModeOff = "off"

	StateHeat = "heat"
	StateIdle = "idle"
)

// Controller is an interface representing an actual climate control device.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Controller interface {
	// SetThermostatMode sets a new thermostat mode.
	SetThermostatMode(mode string) error
	// SetThermostatSetpoint sets a setpoint for a particular mode.
	SetThermostatSetpoint(mode string, value float64, unit string) error
	// ThermostatModeReport returns a current mode information.
	ThermostatModeReport() (mode string, err error)
	// ThermostatSetpointReport returns a current setpoint for given mode.
	ThermostatSetpointReport(mode string) (value float64, unit string, err error)
	// ThermostatStateReport returns a current state of the thermostat.
	ThermostatStateReport() (string, error)
}

// Service is an interface representing a thermostat FIMP service.
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
	// SupportedModes returns modes that are supported by the thermostat.
	SupportedModes() []string
	// SupportedSetpoints returns setpoints that are supported by the thermostat.
	SupportedSetpoints() []string
	// SupportedStates returns states that are supported by the thermostat.
	SupportedStates() []string
}

// NewService creates new instance of a thermostat FIMP service.
func NewService(
	mqtt *fimpgo.MqttTransport,
	specification *fimptype.Service,
	controller Controller,
) Service {
	specification.Name = Thermostat

	return &service{
		Service:    adapter.NewService(mqtt, specification),
		controller: controller,
	}
}

// service is a private implementation of a thermostat FIMP service.
type service struct {
	adapter.Service

	controller Controller
}

// SetMode sets mode of the device.
func (s *service) SetMode(mode string) error {
	normalizedMode, ok := s.normalizeMode(mode)
	if !ok {
		return fmt.Errorf("%s: mode is unsupported: %s", s.Name(), mode)
	}

	err := s.controller.SetThermostatMode(normalizedMode)
	if err != nil {
		return fmt.Errorf("%s: failed to set mode %s: %w", s.Name(), normalizedMode, err)
	}

	return nil
}

// SetSetpoint sets setpoint for a specific mode. Unit value is ignored and maintained for informational purpose only.
func (s *service) SetSetpoint(mode string, value float64, unit string) error {
	normalizedMode, ok := s.normalizeMode(mode)
	if !ok {
		return fmt.Errorf("%s: mode is unsupported: %s", s.Name(), mode)
	}

	err := s.controller.SetThermostatSetpoint(normalizedMode, value, unit)
	if err != nil {
		return fmt.Errorf("%s: failed to set setpoint for mode %s for value %.01f: %w", s.Name(), normalizedMode, value, err)
	}

	return nil
}

// SendModeReport sends a mode report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendModeReport(_ bool) (bool, error) {
	value, err := s.controller.ThermostatModeReport()
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
	normalizedMode, ok := s.normalizeMode(mode)
	if !ok {
		return false, fmt.Errorf("%s: mode is unsupported: %s", s.Name(), mode)
	}

	value, unit, err := s.controller.ThermostatSetpointReport(normalizedMode)
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve setpoint report for mode %s: %w", s.Name(), normalizedMode, err)
	}

	message := fimpgo.NewStrMapMessage(
		EvtSetpointReport,
		s.Name(),
		NewSetpoint(normalizedMode, value, unit).StringMap(),
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
	value, err := s.controller.ThermostatStateReport()
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

// SupportedModes returns modes that are supported by the thermostat.
func (s *service) SupportedModes() []string {
	return s.Service.Specification().PropertyStrings("sup_modes")
}

// SupportedSetpoints returns setpoints that are supported by the thermostat.
func (s *service) SupportedSetpoints() []string {
	return s.Service.Specification().PropertyStrings("sup_setpoints")
}

// SupportedStates returns states that are supported by the thermostat.
func (s *service) SupportedStates() []string {
	return s.Service.Specification().PropertyStrings("sup_states")
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

// Setpoint is an object representing a Thermostat setpoint.
type Setpoint struct {
	Mode        string
	Temperature float64
	Unit        string
}

// NewSetpoint create a new setpoint object.
func NewSetpoint(mode string, temp float64, unit string) *Setpoint {
	return &Setpoint{
		Mode:        mode,
		Temperature: temp,
		Unit:        unit,
	}
}

// StringMap creates a string map out of existing setpoint object.
func (s *Setpoint) StringMap() map[string]string {
	return map[string]string{
		"mode": s.Mode,
		"temp": strconv.FormatFloat(s.Temperature, 'f', 1, 64),
		"unit": s.Unit,
	}
}

// SetpointFromStringMap converts string map into a Setpoint object.
func SetpointFromStringMap(input map[string]string) (*Setpoint, error) {
	mode, ok := input["mode"]
	if !ok {
		return nil, fmt.Errorf("setpoint: missing `mode` field in a string map")
	}

	unit, ok := input["unit"]
	if !ok {
		return nil, fmt.Errorf("setpoint: missing `unit` field in a string map")
	}

	tempStr, ok := input["temp"]
	if !ok {
		return nil, fmt.Errorf("setpoint: missing `temp` field in a string map")
	}

	temp, err := strconv.ParseFloat(tempStr, 64)
	if err != nil {
		return nil, fmt.Errorf("setpoint: cannot parse `temp` field %s: %w", tempStr, err)
	}

	return &Setpoint{
		Mode:        mode,
		Temperature: temp,
		Unit:        unit,
	}, nil
}
