package numericsensor

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

// Constants defining important properties specific for the service.
const (
	UnitC        = "C"
	UnitF        = "F"
	UnitPercent  = "%"
	UnitKph      = "kph"
	UnitMilliBar = "mbar"
	UnitDecibel  = "dB"
	UnitDeg      = "deg"
	UnitPpm      = "ppm"
	UnitMmPerH   = "mm/h"
	UnitPM25     = "pm25"
	UnitPM10     = "pm10"
	UnitAQI      = "aqi"

	PropertyUnit           = "unit"
	PropertySupportedUnits = "sup_units"
)

// DefaultReportingStrategy is the default reporting strategy used by the service for periodic reports.
var DefaultReportingStrategy = cache.ReportAtLeastEvery(30 * time.Minute)

// Reporter is an interface representing an actual device reporting numeric sensor values.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Reporter interface {
	// NumericSensorReport returns numeric sensor report based on the requested unit.
	NumericSensorReport(unit string) (float64, error)
}

// Service is an interface representing a numeric sensor FIMP service.
type Service interface {
	adapter.Service

	// SendSensorReport sends a numeric sensor report based on requested unit. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendSensorReport(unit string, force bool) (bool, error)
	// SupportedUnits returns units that are supported by the numeric sensor report.
	SupportedUnits() []string
}

// Config represents a service configuration.
type Config struct {
	Specification     *fimptype.Service
	Reporter          Reporter
	ReportingStrategy cache.ReportingStrategy
}

// NewService creates new instance of a numeric sensor FIMP service.
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
		sensor:            cfg.Reporter,
		lock:              &sync.Mutex{},
		reportingStrategy: cfg.ReportingStrategy,
		reportingCache:    cache.NewReportingCache(),
	}
}

// service is a private implementation of a numeric sensor FIMP service.
type service struct {
	adapter.Service

	sensor            Reporter
	lock              *sync.Mutex
	reportingCache    cache.ReportingCache
	reportingStrategy cache.ReportingStrategy
}

// SendSensorReport sends a numeric sensor report based on requested unit. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendSensorReport(unit string, force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	normalizedUnit, ok := s.normalizeUnit(unit)
	if !ok {
		return false, fmt.Errorf("%s: unit is unsupported: %s", s.Name(), unit)
	}

	value, err := s.sensor.NumericSensorReport(unit)
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve sensor report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtSensorReport, normalizedUnit, value) {
		return false, nil
	}

	message := fimpgo.NewFloatMessage(
		EvtSensorReport,
		s.Name(),
		value,
		map[string]string{
			PropertyUnit: normalizedUnit,
		},
		nil,
		nil,
	).WithStorageStrategy(fimpgo.StorageStrategyAggregate, normalizedUnit)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send sensor report for unit %s: %w", s.Name(), normalizedUnit, err)
	}

	s.reportingCache.Reported(EvtSensorReport, normalizedUnit, value)

	return true, nil
}

// SupportedUnits returns units that are supported by the numeric sensor report.
func (s *service) SupportedUnits() []string {
	return s.Service.Specification().PropertyStrings(PropertySupportedUnits)
}

// normalizeUnit checks if unit is supported and returns its normalized form.
func (s *service) normalizeUnit(unit string) (string, bool) {
	for _, u := range s.SupportedUnits() {
		if strings.EqualFold(unit, u) {
			return u, true
		}
	}

	return "", false
}
