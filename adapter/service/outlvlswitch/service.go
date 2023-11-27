package outlvlswitch

import (
	"fmt"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/pkg/errors"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
	"github.com/futurehomeno/cliffhanger/utils"
)

// Constants defining important properties specific for the service.
const (
	PropertyMaxLvl     = "max_lvl"
	PropertyMinLvl     = "min_lvl"
	PropertySwitchType = "sw_type" // "on_off" or "up_down"

	PropertySupportDuration   = "sup_duration"
	PropertySupportStartLevel = "sup_start_lvl"

	SwitchTypeOnAndOff  = "on_off"
	SwitchTypeUpAndDown = "up_down"

	Duration = "duration"
	StartLvl = "start_lvl"

	TransitionUp   = "up"
	TransitionDown = "down"
)

// DefaultReportingStrategy is the default reporting strategy used by the service for periodic reports.
var DefaultReportingStrategy = cache.ReportOnChangeOnly()

// LevelTransitionParams keeps all properties of the transition controller.
// nil value of the field means the property isn't supported.
type LevelTransitionParams struct {
	StartLvl *int
	Duration *time.Duration
}

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

// LevelTransitionController represents a controller over a single device for level transitioning.
type LevelTransitionController interface {
	// StartLevelTransition starts a transition. Supported values are: "up" and "down"
	StartLevelTransition(string, LevelTransitionParams) error
	// StopLevelTransition stops a transition
	StopLevelTransition() error
}

// Service is an interface representing a output level switch FIMP service.
type Service interface {
	adapter.Service

	// SendLevelReport sends a level report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendLevelReport(force bool) (bool, error)
	// SetLevel sets a level value.
	SetLevel(value int64, duration *time.Duration) error
	// SetBinaryState sets a binary value.
	SetBinaryState(value bool) error
	// StartLevelTransition starts a transition. Supported values are: "up" and "down"
	StartLevelTransition(string, LevelTransitionParams) error
	// StopLevelTransition stops a transition
	StopLevelTransition() error
}

// Config represents a service configuration.
type Config struct {
	Specification     *fimptype.Service
	Controller        Controller
	ReportingStrategy cache.ReportingStrategy
}

// NewService creates new instance of a output level switch FIMP service.
func NewService(
	publisher adapter.ServicePublisher,
	cfg *Config,
) Service {
	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	if cfg.ReportingStrategy == nil {
		cfg.ReportingStrategy = DefaultReportingStrategy
	}

	s := &service{
		Service:           adapter.NewService(publisher, cfg.Specification),
		lock:              &sync.Mutex{},
		controller:        cfg.Controller,
		reportingStrategy: cfg.ReportingStrategy,
		reportingCache:    cache.NewReportingCache(),
	}

	if s.supportsLevelTransition() {
		s.Specification().EnsureInterfaces(levelTransitionInterfaces()...)
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
func (s *service) SetLevel(value int64, duration *time.Duration) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if duration == nil {
		duration = utils.Ptr(time.Duration(0))
	}

	err := s.controller.SetLevelSwitchLevel(value, *duration)
	if err != nil {
		return fmt.Errorf("%s: failed to set level: %w", s.Name(), err)
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

// StartLevelTransition implements starting of the transition with validations and concurrent safety.
func (s *service) StartLevelTransition(value string, params LevelTransitionParams) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.supportsLevelTransition() {
		return fmt.Errorf("level transition isn't supported, can't start")
	}

	if value != TransitionUp && value != TransitionDown {
		return fmt.Errorf("received incorrect value to start level transition. Received: %s Supported: %s, %s", value, TransitionUp, TransitionDown)
	}

	err := s.validateStartLevelOption(params.StartLvl)
	if err != nil {
		return errors.Wrap(err, "validation of the start_lvl property has failed")
	}

	if !s.Specification().PropertyBool(PropertySupportStartLevel) {
		params.StartLvl = nil
	}

	if !s.Specification().PropertyBool(PropertySupportDuration) {
		params.Duration = nil
	}

	ctr, ok := s.controller.(LevelTransitionController)
	if !ok {
		return fmt.Errorf("failed to cast controller into LevelTransitionController when starting level transition")
	}

	if err := ctr.StartLevelTransition(value, params); err != nil {
		return errors.Wrap(err, "failed to start level transition")
	}

	return nil
}

// StopLevelTransition implements stopping of the transition with validations and concurrent safety.
func (s *service) StopLevelTransition() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.supportsLevelTransition() {
		return fmt.Errorf("level transition isn't supported, can't stop")
	}

	ctr, ok := s.controller.(LevelTransitionController)
	if !ok {
		return fmt.Errorf("failed to cast controller into LevelTransitionController when stopping level transition")
	}

	if err := ctr.StopLevelTransition(); err != nil {
		return errors.Wrap(err, "failed to start level transition")
	}

	return nil
}

func (s *service) supportsLevelTransition() bool {
	_, ok := s.controller.(LevelTransitionController)

	return ok
}

func (s *service) validateStartLevelOption(startLvl *int) error {
	if startLvl == nil {
		return nil
	}

	lvlMax, ok := s.Specification().PropertyInteger(PropertyMaxLvl)
	if !ok {
		return fmt.Errorf("invalid service specification property: %s should be int", PropertyMaxLvl)
	}

	lvlMin, ok := s.Specification().PropertyInteger(PropertyMinLvl)
	if !ok {
		return fmt.Errorf("invalid service specification property: %s should be int", PropertyMinLvl)
	}

	if *startLvl < int(lvlMin) || int(lvlMax) < *startLvl {
		return fmt.Errorf("invalid startLvl received: %d. Should be in range: %d - %d", startLvl, lvlMin, lvlMax)
	}

	return nil
}
