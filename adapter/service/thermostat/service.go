package sensornumeric

import (
	"github.com/futurehomeno/cliffhanger/adapter"
)

// Thermostat is an interface representing an actual climate control device.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Thermostat interface {
	SetThermostatMode(mode string) error
	SetThermostatSetpoint(mode string, value float64, unit string) error
	ThermostatMode() (mode string, err error)
	ThermostatSetpoint(mode string) (value float64, unit string, err error)
	ThermostatState() (string, error)
}

// Service is an interface representing a thermostat FIMP service.
type Service interface {
	adapter.Service

	SetMode(mode string) error
	SetSetpoint(mode string, value float64, unit string) error
	SendModeReport(force bool) (bool, error)
	SendSetpointReport(mode string, force bool) (bool, error)
	SendStateReport(force bool) (bool, error)
	// SupportedUnits returns units that are supported by the numeric sensor report.
	SupportedUnits() []string
}
