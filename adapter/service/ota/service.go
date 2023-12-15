package ota

import (
	"fmt"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
)

// Controller is an interface representing a device capable of OTA firmware update.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Controller interface {
	// StartOTAUpdate starts an OTA update process with provided firmware path.
	StartOTAUpdate(firmwarePath string) error

	// OTAStatusReport returns an OTA update report.
	OTAStatusReport() (StatusReport, error)
}

// Service is an interface representing an OTA FIMP service.
type Service interface {
	adapter.Service

	// StartUpdate starts an OTA update process with provided firmware path.
	StartUpdate(firmwarePath string) error

	// SendStatusReport sends an OTA status report.
	// If controller reports StatusIdle, no report is sent.
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

	return nil
}

func (s *service) SendStatusReport() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	report, err := s.controller.OTAStatusReport()
	if err != nil {
		return fmt.Errorf("failed to get OTA status report: %w", err)
	}

	if err = report.validate(); err != nil {
		return fmt.Errorf("invalid OTA update report: %w", err)
	}

	var message *fimpgo.FimpMessage

	switch report.Status {
	case StatusIdle:
		return nil
	case StatusStarted:
		message = s.newStartReport()
	case StatusInProgress:
		message = s.newProgressReport(report.Progress)
	case StatusDone:
		message = s.newEndReport(report.Result)
	}

	if err = s.SendMessage(message); err != nil {
		return fmt.Errorf("failed to send OTA status report: %w", err)
	}

	return nil
}

func (s *service) newStartReport() *fimpgo.FimpMessage {
	return fimpgo.NewNullMessage(
		EvtOTAStartReport,
		s.Name(),
		nil,
		nil,
		nil,
	)
}

func (s *service) newProgressReport(data ProgressData) *fimpgo.FimpMessage {
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
		s.Name(),
		value,
		nil,
		nil,
		nil,
	)
}

func (s *service) newEndReport(data ResultData) *fimpgo.FimpMessage {
	value := EndReport{
		Success: data.Error == "",
		Error:   data.Error.String(),
	}

	return fimpgo.NewObjectMessage(
		EvtOTAEndReport,
		s.Name(),
		value,
		nil,
		nil,
		nil,
	)
}
