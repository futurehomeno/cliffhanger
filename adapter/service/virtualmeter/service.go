package virtualmeter

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

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

		SendModesReport(bool) (bool, error)
		AddMeter(map[string]float64, string) error
		RemoveMeter() error
	}

	Config struct {
		Specification       *fimptype.Service
		VirtualMeterManager Manager
		ReportingStrategy   cache.ReportingStrategy
	}

	service struct {
		adapter.Service

		manager           Manager
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
		cfg.ReportingStrategy = cache.ReportAtLeastEvery(time.Minute * 30)
	}

	s := &service{
		Service:           adapter.NewService(publisher, cfg.Specification),
		manager:           cfg.VirtualMeterManager,
		lock:              &sync.RWMutex{},
		reportingCache:    cache.NewReportingCache(),
		reportingStrategy: cfg.ReportingStrategy,
	}

	return s
}

func (s *service) SendModesReport(force bool) (bool, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	value, err := s.manager.Modes(s.Specification().Address)
	if err != nil {
		return false, fmt.Errorf("failed to send virtual meter report: %w", err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtMeterReport, "", value) {
		return false, nil
	}

	log.Infof("Sending modes report: %v, name: %s, ", value, s.Name())

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
		return false, fmt.Errorf("%s - failed to send virtual meter report: %w", s.Name(), err)
	}

	s.reportingCache.Reported(EvtMeterReport, "", value)

	return true, nil
}

func (s *service) AddMeter(modes map[string]float64, unit string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	supportedUnits := s.Specification().PropertyStrings(PropertySupportedUnits)
	if !slices.Contains(supportedUnits, unit) {
		return fmt.Errorf("%s: unsupported unit is provided: %s. Supported: %v", s.Name(), unit, supportedUnits)
	}

	for mode := range modes {
		if !slices.Contains(s.Specification().PropertyStrings(PropertySupportedModes), mode) {
			log.Infof("Provided unsupported mode: %s. Removing. Supported: %v", mode, s.Specification().PropertyStrings(PropertySupportedModes))
			delete(modes, mode)
		}
	}

	return s.manager.Add(s.Specification().Address, modes, unit)
}

func (s *service) RemoveMeter() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if err := s.manager.Remove(s.Specification().Address); err != nil {
		return fmt.Errorf("failed to remove meter: %w", err)
	}

	return nil
}
