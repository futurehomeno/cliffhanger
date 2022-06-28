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
	PropertyMaxLvl     = "max_lvl"
	PropertyMinLvl     = "min_lvl"
	PropertySwitchType = "sw_type" // "on_off" or "up_down"

	SwitchTypeOnAndOff  = "on_off"
	SwitchTypeUpAndDown = "up_down"

	Duration = "duration"
)

// DefaultReportingStrategy is the default reporting strategy used by the service for periodic reports.
var DefaultReportingStrategy = cache.ReportOnChangeOnly()

// Controller is an interface representing an actual device.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Controller interface {
	// LevelSwitchLevelReport returns a current level value.
	LevelSwitchLevelReport() (int64, error)
	// SetLevelSwitchLevel sets a level value.
	SetLevelSwitchLevel(value int64, duration time.Duration) error
	// SetLevelSwitchBinaryState sets a binary value.
	SetLevelSwitchBinaryState(bool) error
}

// Service is an interface representing a output level switch FIMP service.
type Service interface {
	adapter.Service

	// SendLevelReport sends a level report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendLevelReport(force bool) (bool, error)
	// SetLevel sets a level value.
	SetLevel(value int64, duration time.Duration) error
	// SetBinaryState sets a binary value.
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

	s := &service{
		Service:           adapter.NewService(mqtt, cfg.Specification),
		lock:              &sync.Mutex{},
		controller:        cfg.Controller,
		reportingStrategy: cfg.ReportingStrategy,
		reportingCache:    cache.NewReportingCache(),
	}

	return s
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

// SetLevel sets a level value.
func (s *service) SetLevel(value int64, duration time.Duration) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if duration.Nanoseconds() > 0 {
		err := s.controller.SetLevelSwitchLevel(value, duration)
		if err != nil {
			return fmt.Errorf("%s: failed to set level: %w", s.Name(), err)
		}
	} else {
		err := s.controller.SetLevelSwitchLevel(value, time.Duration(0))
		if err != nil {
			return fmt.Errorf("%s: failed to set level: %w", s.Name(), err)
		}
	}

	return nil
}

// SetBinaryState sets a binary value.
func (s *service) SetBinaryState(value bool) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := s.controller.SetLevelSwitchBinaryState(value)
	if err != nil {
		return fmt.Errorf("%s: failed to set binary: %w", s.Name(), err)
	}

	return nil
}
