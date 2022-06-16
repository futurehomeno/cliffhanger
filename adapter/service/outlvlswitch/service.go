package outlvlswitch

import (
	"fmt"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

// Constants defining important properties specific for the service.
const (
	MaxLvl        = "max_lvl"
	MinLvl        = "min_lvl"
	SwitchType    = "sw_type" // "on_off" or "up_down"
	TypeOnAndOff  = "on_off"
	TypeUpAndDown = "up_down"
)

// DefaultReportingStrategy is the default reporting strategy used by the service for periodic reports.
var DefaultReportingStrategy = cache.ReportAtLeastEvery(30 * time.Minute)

// Controller is an interface representing an actual device.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Controller interface {
	// LevelReport returns a current level value.
	LevelSwitchLevelReport() (int64, error)
	// BinanryReport returns a current binary value.
	LevelSwitchBinaryReport() (bool, error)
	// SetLvlCtrl sets a level value.
	SetLevelSwitchLevel(value int64) error
	// SetLevelWithDurationCtrl sets a level value over a specified duration in seconds.
	SetLevelSwitchLevelWithDuration(value int64, duration int64) error
	// SetBinaryCtrl sets a binary value.
	SetLevelSwitchBinaryState(bool) error
}

// Service is an interface representing a output level switch FIMP service.
type Service interface {
	adapter.Service

	// SendLevelReport sends a level report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendLevelReport(force bool) (bool, error)
	// SendBinaryReport sends a binary report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendBinaryReport(force bool) (bool, error)
	// SetLevel sets a level value.
	SetLevel(value int64) error
	// SetLevelWithDuration sets a level value over a specified duration in seconds.
	SetLevelWithDuration(value int64, duration int64) error
	// SetBinary sets a binary value.
	SetBinaryState(value bool) error
}

// Config represents a service configuration.
type Config struct {
	Specification     *fimptype.Service
	Controller        Controller
	ReportingStrategy cache.ReportingStrategy
}

// NewService creates new instance of a output level switch FIMP service.
func NewService(
	mqtt *fimpgo.MqttTransport,
	cfg *Config,
) Service {
	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	if cfg.ReportingStrategy == nil {
		cfg.ReportingStrategy = DefaultReportingStrategy
	}

	return &service{
		Service:           adapter.NewService(mqtt, cfg.Specification),
		controller:        cfg.Controller,
		lock:              &sync.Mutex{},
		reportingStrategy: cfg.ReportingStrategy,
		reportingCache:    cache.NewReportingCache(),
	}
}

// service is a private implementation of a output level switch FIMP service.
type service struct {
	adapter.Service

	controller        Controller
	lock              *sync.Mutex
	reportingCache    cache.ReportingCache
	reportingStrategy cache.ReportingStrategy
}

// SendLevelReport sends a level report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendLevelReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	value, err := s.controller.LevelSwitchLevelReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to get level report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtLvlReport, "", value) {
		return false, nil
	}

	message := fimpgo.NewIntMessage(
		EvtLvlReport,
		s.Name(),
		value,
		nil,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send level report: %w", s.Name(), err)
	}

	s.reportingCache.Reported(EvtLvlReport, "", value)

	return true, nil
}

// SendBinaryReport sends a binary report. Returns true if a report was sent.
func (s *service) SendBinaryReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	value, err := s.controller.LevelSwitchBinaryReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to get binary report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtBinaryReport, "", value) {
		return false, nil
	}

	message := fimpgo.NewBoolMessage(
		EvtBinaryReport,
		s.Name(),
		value,
		nil,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send binary report: %w", s.Name(), err)
	}

	s.reportingCache.Reported(EvtBinaryReport, "", value)

	return true, nil
}

// SetLevel sets a level value.
func (s *service) SetLevel(value int64) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := s.controller.SetLevelSwitchLevel(value)
	if err != nil {
		return fmt.Errorf("%s: failed to set level: %w", s.Name(), err)
	}

	return nil
}

// SetLevelWithDuration sets a level value over a specified duration.
func (s *service) SetLevelWithDuration(value int64, duration int64) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := s.controller.SetLevelSwitchLevelWithDuration(value, duration)
	if err != nil {
		return fmt.Errorf("%s: failed to set level with duration: %w", s.Name(), err)
	}

	return nil
}

// SetBinary sets a binary value.
func (s *service) SetBinaryState(value bool) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := s.controller.SetLevelSwitchBinaryState(value)
	if err != nil {
		return fmt.Errorf("%s: failed to set binary: %w", s.Name(), err)
	}

	return nil
}
