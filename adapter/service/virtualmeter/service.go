package virtualmeter

import (
	"fmt"
	"slices"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

const (
	PropertySupportedUnits = "sup_units"
	PropertySupportedModes = "sup_modes"

	ModeOff = "off"
	ModeOn  = "on"
)

type (
	Service interface {
		adapter.Service

		// SendModesReport sends a report on the current modes of the virtual meter.
		SendModesReport(bool) (bool, error)
		// AddMeter updates a device by this service with a new pair of modes and unit,
		// replacing existing or addining the first one.
		AddMeter(map[string]float64, string) error
		// RemoveMeter removes set modes and unit for the device.
		RemoveMeter() error
	}

	Config struct {
		Specification     *fimptype.Service
		Manager           Manager
		ReportingStrategy cache.ReportingStrategy
	}

	service struct {
		adapter.Service

		manager           *manager
		lock              *sync.RWMutex
		reportingCache    cache.ReportingCache
		reportingStrategy cache.ReportingStrategy
	}
)

func NewService(
	publisher adapter.ServicePublisher,
	cfg *Config,
) Service {
	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	if cfg.ReportingStrategy == nil {
		cfg.ReportingStrategy = cache.ReportOnChangeOnly()
	}

	mr := cfg.Manager.(*manager) //nolint:forcetypeassert

	s := &service{
		Service:           adapter.NewService(publisher, cfg.Specification),
		manager:           mr,
		lock:              &sync.RWMutex{},
		reportingCache:    cache.NewReportingCache(),
		reportingStrategy: cfg.ReportingStrategy,
	}

	return s
}

func (s *service) SendModesReport(force bool) (bool, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	value, err := s.manager.modes(s.Specification().Address)
	if err != nil {
		return false, fmt.Errorf("service: failed to send virtual meter report: %w", err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtMeterReport, "", value) {
		return false, nil
	}

	sendValue := value
	if sendValue == nil {
		sendValue = make(map[string]float64)
	}

	message := fimpgo.NewFloatMapMessage(
		EvtMeterReport,
		s.Name(),
		sendValue,
		nil,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("service: %s - failed to send virtual meter report: %w", s.Name(), err)
	}

	s.reportingCache.Reported(EvtMeterReport, "", value)

	return true, nil
}

func (s *service) AddMeter(modes map[string]float64, unit string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	supportedUnits := s.Specification().PropertyStrings(PropertySupportedUnits)
	if !slices.Contains(supportedUnits, unit) {
		return fmt.Errorf("service: %s: unsupported unit is provided: %s. Supported: %v", s.Name(), unit, supportedUnits)
	}

	if len(modes) != len(s.Specification().PropertyStrings(PropertySupportedModes)) {
		return fmt.Errorf("service: %s: number of modes is not equal to the number of supported modes", s.Name())
	}

	for mode := range modes {
		if !slices.Contains(s.Specification().PropertyStrings(PropertySupportedModes), mode) {
			return fmt.Errorf("service: %s: unsupported mode is provided: %s. Supported: %v", s.Name(), mode, s.Specification().PropertyStrings(PropertySupportedModes))
		}
	}

	return s.manager.add(s.Specification().Address, modes, unit)
}

func (s *service) RemoveMeter() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if err := s.manager.remove(s.Specification().Address); err != nil {
		return fmt.Errorf("service: failed to remove meter: %w", err)
	}

	return nil
}
