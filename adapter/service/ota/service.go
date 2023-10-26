package ota

import (
	"fmt"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
)

// Controller is an interface representing a device capable of OTA firmware updated.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Controller interface {
	// StartOTAUpdate starts an OTA update process with provided firmware path.
	StartOTAUpdate(firmwarePath string) error

	// OTAUpdateReport returns an OTA update report.
	OTAUpdateReport() (UpdateReport, error)
}

// Service is an interface representing an OTA FIMP service.
type Service interface {
	adapter.Service

	// StartUpdate starts an OTA update process with provided firmware path.
	StartUpdate(firmwarePath string) error

	// SendStatusReport sends an OTA status report.
	// If controller reports idle status, no report is sent.
	SendStatusReport() error
}

// Config represents a service configuration.
type Config struct {
	Specification *fimptype.Service
	Controller    Controller
}

// NewService creates a new instance of an OTA FIMP service.
func NewService(
	publisher adapter.ServicePublisher,
	cfg *Config,
) Service {
	cfg.Specification.Name = OTA

	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	s := &service{
		Service:    adapter.NewService(publisher, cfg.Specification),
		controller: cfg.Controller,
		lock:       &sync.Mutex{},
	}

	return s
}

type service struct {
	adapter.Service

	controller Controller
	lock       *sync.Mutex
}

func (s *service) StartUpdate(firmwarePath string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if firmwarePath == "" {
		return fmt.Errorf("firmware path is empty")
	}

	if err := s.controller.StartOTAUpdate(firmwarePath); err != nil {
		return fmt.Errorf("failed to start OTA update: %w", err)
	}

	message := fimpgo.NewNullMessage(
		EvtOTAStartReport,
		s.Name(),
		nil,
		nil,
		nil,
	)

	if err := s.SendMessage(message); err != nil {
		return fmt.Errorf("failed to send OTA start report: %w", err)
	}

	return nil
}

func (s *service) SendStatusReport() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	report, err := s.controller.OTAUpdateReport()
	if err != nil {
		return fmt.Errorf("failed to get OTA update report: %w", err)
	}

	if err = report.validate(); err != nil {
		return fmt.Errorf("invalid OTA update report: %w", err)
	}

	var message *fimpgo.FimpMessage

	switch report.Status {
	case StatusIdle:
		return nil
	case StatusInProgress:
		message = newProgressReport(report.Progress)
	case StatusDone:
		message = newEndReport(report.Result)
	}

	if err = s.SendMessage(message); err != nil {
		return fmt.Errorf("failed to send OTA status report: %w", err)
	}

	return nil
}

func newProgressReport(data ProgressData) *fimpgo.FimpMessage {
	value := map[string]int64{
		"progress": int64(data.Progress),
	}

	if data.RemainingMinutes > 0 {
		value["remaining_min"] = int64(data.RemainingMinutes)
	}

	if data.RemainingSeconds > 0 {
		value["remaining_sec"] = int64(data.RemainingSeconds)
	}

	return fimpgo.NewIntMapMessage(
		EvtOTAProgressReport,
		OTA,
		value,
		nil,
		nil,
		nil,
	)
}

func newEndReport(data ResultData) *fimpgo.FimpMessage {
	value := EndReport{
		Success: data.Error == "",
		Error:   data.Error.String(),
	}

	return fimpgo.NewObjectMessage(
		EvtOTAEndReport,
		OTA,
		value,
		nil,
		nil,
		nil,
	)
}
