package colorctrl

import (
	"fmt"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

const (
	PropertySupportedComponents = "sup_components"
	PropertySupportedDurations  = "sup_durations"
)

// DefaultReportingStrategy is the default reporting strategy used by the service for periodic reports of state changes.
var DefaultReportingStrategy = cache.ReportOnChangeOnly()

// Controller is an interface representing an actual device using color service.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Controller interface {
	// SetColorCtrlColor sets the color of the device.
	SetColorCtrlColor(color map[string]int64) error
	// ColorCtrlColorReport returns a current color of the device.
	ColorCtrlColorReport() (map[string]int64, error)
}

// Service is an interface representing a colorctrl FIMP service.
type Service interface {
	adapter.Service

	// SetColor sets the color of the device.
	SetColor(color map[string]int64) error
	// SendColorReport sends a current color report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendColorReport(force bool) (bool, error)
	// SupportedComponents returns a list of supported color components.
	SupportedComponents() []string
	// SupportedDurations returns a list of supported durations.
	SupportedDurations() []int64
}

// Config represents a service configuration.
type Config struct {
	Specification     *fimptype.Service
	Controller        Controller
	ReportingStrategy cache.ReportingStrategy
}

// NewService creates a new instance of a colorctrl FIMP service.
func NewService(
	publisher adapter.ServicePublisher,
	cfg *Config,
) Service {
	cfg.Specification.Name = ColorCtrl

	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	if cfg.ReportingStrategy == nil {
		cfg.ReportingStrategy = DefaultReportingStrategy
	}

	s := &service{
		Service:           adapter.NewService(publisher, cfg.Specification),
		controller:        cfg.Controller,
		lock:              &sync.Mutex{},
		reportingCache:    cache.NewReportingCache(),
		reportingStrategy: cfg.ReportingStrategy,
	}

	return s
}

// service is a private implementation of a colorctrl FIMP service.
type service struct {
	adapter.Service

	controller        Controller
	lock              *sync.Mutex
	reportingCache    cache.ReportingCache
	reportingStrategy cache.ReportingStrategy
}

// SetColor sets the color of the device.
func (s *service) SetColor(color map[string]int64) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := s.controller.SetColorCtrlColor(color)
	if err != nil {
		return fmt.Errorf("failed to set color: %w", err)
	}

	return nil
}

// SendColorReport sends a current color report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendColorReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	color, err := s.controller.ColorCtrlColorReport()
	if err != nil {
		return false, fmt.Errorf("failed to get color: %w", err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtColorReport, "", color) {
		return false, nil
	}

	message := fimpgo.NewIntMapMessage(
		EvtColorReport,
		s.Name(),
		color,
		nil,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("failed to send color report: %w", err)
	}

	s.reportingCache.Reported(EvtColorReport, "", color)

	return true, nil
}

// SupportedComponents returns a list of supported color components.
func (s *service) SupportedComponents() []string {
	return s.Service.Specification().PropertyStrings(PropertySupportedComponents)
}

// SupportedDurations returns a list of supported durations.
func (s *service) SupportedDurations() []int64 {
	return s.Service.Specification().PropertyIntegers(PropertySupportedDurations)
}
