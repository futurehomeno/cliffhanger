package chargepoint

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
)

// Controller is an interface representing an actual car charger device.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Controller interface {
	// StartChargepointCharging starts car charging.
	StartChargepointCharging() error
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
	StartCharging() error
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

// NewService creates new instance of a water heater FIMP service.
func NewService(
	mqtt *fimpgo.MqttTransport,
	specification *fimptype.Service,
	controller Controller,
) Service {
	specification.Name = Chargepoint

	return &service{
		Service:    adapter.NewService(mqtt, specification),
		controller: controller,
	}
}

// service is a private implementation of a water heater FIMP service.
type service struct {
	adapter.Service

	controller Controller
}

// StartCharging starts car charging.
func (s *service) StartCharging() error {
	err := s.controller.StopChargepointCharging()
	if err != nil {
		return fmt.Errorf("%s: failed to start charging: %w", s.Name(), err)
	}

	return nil
}

// StopCharging stops car charging.
func (s *service) StopCharging() error {
	err := s.controller.StopChargepointCharging()
	if err != nil {
		return fmt.Errorf("%s: failed to stop charging: %w", s.Name(), err)
	}

	return nil
}

// SetCableLock locks and unlocks the cable connector.
func (s *service) SetCableLock(lock bool) error {
	err := s.controller.StopChargepointCharging()
	if err != nil {
		return fmt.Errorf("%s: failed to set cable lock to %t: %w", s.Name(), lock, err)
	}

	return nil
}

// SendCurrentSessionReport sends a current charging session report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendCurrentSessionReport(_ bool) (bool, error) {
	value, err := s.controller.ChargepointCurrentSessionReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve current session report: %w", s.Name(), err)
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

	return true, nil
}

// SendCableLockReport sends a cable lock report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendCableLockReport(_ bool) (bool, error) {
	value, err := s.controller.ChargepointCableLockReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve cable lock report: %w", s.Name(), err)
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

	return true, nil
}

// SendStateReport sends a state report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendStateReport(_ bool) (bool, error) {
	value, err := s.controller.ChargepointStateReport()
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

// SupportedStates returns states that are supported by the chargepoint.
func (s *service) SupportedStates() []string {
	return s.Service.Specification().PropertyStrings("sup_states")
}
