package diagnostic

import (
	"fmt"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
)

type Controller any

type LQIReporter interface {
	LQIReport() (int, error)
}

type RSSIReporter interface {
	RSSIReport() (int, error)
}

type RebootReasonReporter interface {
	RebootReasonReport() (string, error)
}

type RebootsCountReporter interface {
	RebootsCountReport() (int, error)
}

type UptimeReporter interface {
	UptimeReport() (int, error)
}

type ErrorsReporter interface {
	ErrorsReport() ([]string, error)
}

type Service interface {
	adapter.Service

	SendLQIReport() error
	SendRSSIReport() error
	SendRebootReasonReport() error
	SendRebootsCountReport() error
	SendUptimeReport() error
	SendErrorsReport() error

	SupportsLQI() bool
	SupportsRSSI() bool
	SupportsRebootReason() bool
	SupportsRebootsCount() bool
	SupportsUptime() bool
	SupportsErrors() bool
}

type Config struct {
	Specification *fimptype.Service
	Controller    Controller
}

func NewService(
	publisher adapter.ServicePublisher,
	cfg *Config,
) Service {
	cfg.Specification.Name = Diagnostic

	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	s := &service{
		Service:    adapter.NewService(publisher, cfg.Specification),
		controller: cfg.Controller,
		lock:       &sync.Mutex{},
	}

	if s.SupportsLQI() {
		cfg.Specification.EnsureInterfaces(lqiInterfaces()...)
	}

	if s.SupportsRSSI() {
		cfg.Specification.EnsureInterfaces(rssiInterfaces()...)
	}

	if s.SupportsRebootReason() {
		cfg.Specification.EnsureInterfaces(rebootReasonInterfaces()...)
	}

	if s.SupportsRebootsCount() {
		cfg.Specification.EnsureInterfaces(rebootsCountInterfaces()...)
	}

	if s.SupportsUptime() {
		cfg.Specification.EnsureInterfaces(uptimeInterfaces()...)
	}

	if s.SupportsErrors() {
		cfg.Specification.EnsureInterfaces(errorsInterfaces()...)
	}

	return s
}

type service struct {
	adapter.Service

	controller Controller
	lock       *sync.Mutex
}

func (s *service) SendLQIReport() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	controller, ok := s.controller.(LQIReporter)
	if !ok {
		return fmt.Errorf("%s: LQI reporting is not supported", s.Name())
	}

	value, err := controller.LQIReport()
	if err != nil {
		return fmt.Errorf("%s: failed to retrieve LQI report: %w", s.Name(), err)
	}

	if err := s.SendMessage(fimpgo.NewIntMessage(EvtLQIReport, s.Name(), value, nil, nil, nil)); err != nil {
		return fmt.Errorf("%s: failed to send LQI report: %w", s.Name(), err)
	}

	return nil
}

func (s *service) SendRSSIReport() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	controller, ok := s.controller.(RSSIReporter)
	if !ok {
		return fmt.Errorf("%s: RSSI reporting is not supported", s.Name())
	}

	value, err := controller.RSSIReport()
	if err != nil {
		return fmt.Errorf("%s: failed to retrieve RSSI report: %w", s.Name(), err)
	}

	if err := s.SendMessage(fimpgo.NewIntMessage(EvtRSSIReport, s.Name(), value, nil, nil, nil)); err != nil {
		return fmt.Errorf("%s: failed to send RSSI report: %w", s.Name(), err)
	}

	return nil
}

func (s *service) SendRebootReasonReport() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	controller, ok := s.controller.(RebootReasonReporter)
	if !ok {
		return fmt.Errorf("%s: reboot reason reporting is not supported", s.Name())
	}

	value, err := controller.RebootReasonReport()
	if err != nil {
		return fmt.Errorf("%s: failed to retrieve reboot reason report: %w", s.Name(), err)
	}

	if err := s.SendMessage(fimpgo.NewStringMessage(EvtRebootReasonReport, s.Name(), value, nil, nil, nil)); err != nil {
		return fmt.Errorf("%s: failed to send reboot reason report: %w", s.Name(), err)
	}

	return nil
}

func (s *service) SendRebootsCountReport() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	controller, ok := s.controller.(RebootsCountReporter)
	if !ok {
		return fmt.Errorf("%s: reboots count reporting is not supported", s.Name())
	}

	value, err := controller.RebootsCountReport()
	if err != nil {
		return fmt.Errorf("%s: failed to retrieve reboots count report: %w", s.Name(), err)
	}

	if err := s.SendMessage(fimpgo.NewIntMessage(EvtRebootCountReport, s.Name(), value, nil, nil, nil)); err != nil {
		return fmt.Errorf("%s: failed to send reboots count report: %w", s.Name(), err)
	}

	return nil
}

func (s *service) SendUptimeReport() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	controller, ok := s.controller.(UptimeReporter)
	if !ok {
		return fmt.Errorf("%s: uptime reporting is not supported", s.Name())
	}

	value, err := controller.UptimeReport()
	if err != nil {
		return fmt.Errorf("%s: failed to retrieve uptime report: %w", s.Name(), err)
	}

	if err := s.SendMessage(fimpgo.NewIntMessage(EvtUptimeReport, s.Name(), value, nil, nil, nil)); err != nil {
		return fmt.Errorf("%s: failed to send uptime report: %w", s.Name(), err)
	}

	return nil
}

func (s *service) SendErrorsReport() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	controller, ok := s.controller.(ErrorsReporter)
	if !ok {
		return fmt.Errorf("%s: errors reporting is not supported", s.Name())
	}

	value, err := controller.ErrorsReport()
	if err != nil {
		return fmt.Errorf("%s: failed to retrieve errors report: %w", s.Name(), err)
	}

	if err := s.SendMessage(fimpgo.NewStrArrayMessage(EvtErrorsReport, s.Name(), value, nil, nil, nil)); err != nil {
		return fmt.Errorf("%s: failed to send errors report: %w", s.Name(), err)
	}

	return nil
}

func (s *service) SupportsLQI() bool {
	_, ok := s.controller.(LQIReporter)

	return ok
}

func (s *service) SupportsRSSI() bool {
	_, ok := s.controller.(RSSIReporter)

	return ok
}

func (s *service) SupportsRebootReason() bool {
	_, ok := s.controller.(RebootReasonReporter)

	return ok
}

func (s *service) SupportsRebootsCount() bool {
	_, ok := s.controller.(RebootsCountReporter)

	return ok
}

func (s *service) SupportsUptime() bool {
	_, ok := s.controller.(UptimeReporter)

	return ok
}

func (s *service) SupportsErrors() bool {
	_, ok := s.controller.(ErrorsReporter)

	return ok
}
