package outlvlswitch

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

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
	SetLevelSwitchLevel(value int64) error
	// SetLevelSwitchBinaryState sets a binary value.
	SetLevelSwitchBinaryState(bool) error
}

type ControllerWithDurationSupport interface {
	Controller

	// SetLevelSwitchLevelWithDuration sets a level value over a specified duration.
	SetLevelSwitchLevelWithDuration(value int64, duration time.Duration) error
}

// Service is an interface representing a output level switch FIMP service.
type Service interface {
	adapter.Service

	// SendLevelReport sends a level report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendLevelReport(force bool) (bool, error)
	// SetLevel sets a level value.
	SetLevel(value int64) error
	// SetLevelWithDuration sets a level value over a specified duration in seconds.
	SetLevelWithDuration(value int64, duration int64) error
	// SetBinaryState sets a binary value.
	SetBinaryState(value bool) error
	// SupportDuration returns true if the service supports duration.
	SupportDuration() bool
}

// Config represents a service configuration.
type Config struct {
	Specification                 *fimptype.Service
	Controller                    Controller
	ControllerWithDurationSupport ControllerWithDurationSupport
	ReportingStrategy             cache.ReportingStrategy
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
		reportingStrategy: cfg.ReportingStrategy,
		reportingCache:    cache.NewReportingCache(),
	}

	s.controller = cfg.Controller
	s.controllerWithDurationSupport = cfg.ControllerWithDurationSupport

	log.Info(reflect.TypeOf(s.controllerWithDurationSupport))
	log.Info("s.controllerWithDurationSupport: ", s.controllerWithDurationSupport) // this logs <nil>
	if s.controllerWithDurationSupport != nil {
		log.Info("supports duration") // but this is true??
	} else {
		log.Info("does not support duration")
	}

	if !s.SupportDuration() {
		log.Info("i am in")
		s.controller = cfg.Controller
		s.controllerWithDurationSupport = nil
	}

	return s
}

// service is a private implementation of a output level switch FIMP service.
type service struct {
	adapter.Service

	controller                    Controller
	controllerWithDurationSupport ControllerWithDurationSupport
	lock                          *sync.Mutex
	reportingCache                cache.ReportingCache
	reportingStrategy             cache.ReportingStrategy
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

	timeDuration := time.Duration(duration) * time.Second

	err := s.controllerWithDurationSupport.SetLevelSwitchLevelWithDuration(value, timeDuration)
	if err != nil {
		return fmt.Errorf("%s: failed to set level with duration: %w", s.Name(), err)
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

// SupportDuration returns true if the service supports duration.
func (s *service) SupportDuration() bool {
	return s.controllerWithDurationSupport != nil
}
