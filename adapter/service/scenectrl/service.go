package scenectrl

import (
	"fmt"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

const (
	PropertySupportedScenes = "sup_scenes"
)

// DefaultReportingStrategy is the default strategy used by the service for periodic reports.
var DefaultReportingStrategy = cache.ReportOnChangeOnly()

// Controller is an interface representing an actual device.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Controller interface {
	// SetSceneCtrlScene sets the scene of the device.
	SetSceneCtrlScene(scene string) error
	// SceneCtrlSceneReport returns the current scene value.
	SceneCtrlSceneReport() (SceneReport, error)
}

type SceneReport struct {
	Scene     string
	Timestamp time.Time
}

// Service is an interface representing a presence FIMP service.
type Service interface {
	adapter.Service

	// SetScene sets the scene of the device.
	SetScene(scene string) error
	// SendSceneReport sends a scene report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendSceneReport(force bool) (bool, error)
}

// Config represents a service configuration.
type Config struct {
	Specification     *fimptype.Service
	Controller        Controller
	ReportingStrategy cache.ReportingStrategy
}

// NewService creates a new instance of a presence FIMP service.
func NewService(
	publisher adapter.ServicePublisher,
	cfg *Config,
) Service {
	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	if cfg.ReportingStrategy == nil {
		cfg.ReportingStrategy = DefaultReportingStrategy
	}

	return &service{
		Service:           adapter.NewService(publisher, cfg.Specification),
		controller:        cfg.Controller,
		lock:              &sync.Mutex{},
		reportingStrategy: cfg.ReportingStrategy,
		reportingCache:    cache.NewReportingCache(),
	}
}

// service is a private implementation of a presence FIMP service.
type service struct {
	adapter.Service

	controller        Controller
	lock              *sync.Mutex
	reportingCache    cache.ReportingCache
	reportingStrategy cache.ReportingStrategy
}

// SetScene sets the scene of the device.
func (s *service) SetScene(scene string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.isSceneSupported(scene) {
		return fmt.Errorf("scene %s is not supported", scene)
	}

	if err := s.controller.SetSceneCtrlScene(scene); err != nil {
		return fmt.Errorf("failed to set scene: %w", err)
	}

	return nil
}

func (s *service) isSceneSupported(scene string) bool {
	supportedScenes := s.Service.Specification().PropertyStrings(PropertySupportedScenes)
	for _, s := range supportedScenes {
		if s == scene {
			return true
		}
	}

	return false
}

// SendSceneReport sends a scene report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendSceneReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	value, err := s.controller.SceneCtrlSceneReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to get scene report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtSceneReport, "", value) {
		return false, nil
	}

	message := fimpgo.NewStringMessage(
		EvtSceneReport,
		s.Name(),
		value.Scene,
		nil,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send scene report: %w", s.Name(), err)
	}

	s.reportingCache.Reported(EvtSceneReport, "", value)

	return true, nil
}
