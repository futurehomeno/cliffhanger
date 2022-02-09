package numericsensor

import (
	"fmt"
	"strings"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
)

// Constants defining important properties specific for the service
const (
	UnitC = "C"
	UnitF = "F"
)

// Reporter is an interface representing an actual device reporting numeric sensor values.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Reporter interface {
	// NumericSensorReport returns numeric sensor report based on the requested unit.
	NumericSensorReport(unit string) (float64, error)
}

// Service is an interface representing a numeric sensor FIMP service.
type Service interface {
	adapter.Service

	// SendSensorReport sends a numeric sensor report based on requested unit. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendSensorReport(unit string, force bool) (bool, error)
	// SupportedUnits returns units that are supported by the numeric sensor report.
	SupportedUnits() []string
}

// NewService creates new instance of a numeric sensor FIMP service.
func NewService(
	mqtt *fimpgo.MqttTransport,
	specification *fimptype.Service,
	reporter Reporter,
) Service {
	return &service{
		Service: adapter.NewService(mqtt, specification),
		sensor:  reporter,
	}
}

// service is a private implementation of a numeric sensor FIMP service.
type service struct {
	adapter.Service

	sensor Reporter
}

// SendSensorReport sends a numeric sensor report based on requested unit. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendSensorReport(unit string, _ bool) (bool, error) {
	normalizedUnit, ok := s.normalizeUnit(unit)
	if !ok {
		return false, fmt.Errorf("%s: unit is unsupported: %s", s.Name(), unit)
	}

	value, err := s.sensor.NumericSensorReport(unit)
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve sensor report: %w", s.Name(), err)
	}

	message := fimpgo.NewFloatMessage(
		EvtSensorReport,
		s.Name(),
		value,
		map[string]string{
			"unit": normalizedUnit,
		},
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send sensor report for unit %s: %w", s.Name(), normalizedUnit, err)
	}

	return true, nil
}

// SupportedUnits returns units that are supported by the numeric sensor report.
func (s *service) SupportedUnits() []string {
	return s.Service.Specification().PropertyStrings("sup_units")
}

// normalizeUnit checks if unit is supported and returns its normalized form.
func (s *service) normalizeUnit(unit string) (string, bool) {
	for _, u := range s.SupportedUnits() {
		if strings.EqualFold(unit, u) {
			return u, true
		}
	}

	return "", false
}
