package virtualmeter

import (
	"fmt"
	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/utils"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	CmdConfigSetInterval    = "cmd.config.set_interval"
	CmdConfigGetInterval    = "cmd.config.get_interval"
	EvtConfigIntervalReport = "evt.config.interval_report"
	CmdMeterAdd             = "cmd.meter.add"
	CmdMeterRemove          = "cmd.meter.remove"
	CmdMeterGetReport       = "cmd.meter.get_report"
	EvtMeterReport          = "evt.meter.report"

	PropertySupportedUnits = "sup_units"
	PropertySupportedModes = "sup_modes"

	TaskVirtualReporter = "virtualReporter"

	ModeOff = "off"
	ModeOn  = "on"
)

type (
	Service interface {
		adapter.Service

		SendReport() error
		AddMeter(map[string]float64, string) error
		RemoveMeter() error

		SetReportingInterval(int) error
		SendReportingInterval() error
	}

	Config struct {
		Specification       *fimptype.Service
		VirtualMeterManager VirtualMeterManager
	}

	service struct {
		adapter.Service

		manager VirtualMeterManager
		lock    *sync.Mutex
	}
)

func NewService(
	publisher adapter.ServicePublisher,
	cfg *Config,
) Service {
	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	s := &service{
		Service: adapter.NewService(publisher, cfg.Specification),
		manager: cfg.VirtualMeterManager,
		lock:    &sync.Mutex{},
	}

	return s
}

func (s *service) SendReport() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	value, err := s.manager.Modes(s.Specification().Address)
	if err != nil {
		return fmt.Errorf("failed to send virtual meter report: %w", err)
	}

	log.Infof("Sending modes report: %v, name: %s, ", value, s.Name())

	if value == nil {
		value = make(map[string]float64)
	}

	message := fimpgo.NewFloatMapMessage(
		EvtMeterReport,
		s.Name(),
		value,
		nil,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return fmt.Errorf("%s - failed to send virtual meter report: %w", s.Name(), err)
	}

	return nil
}

func (s *service) AddMeter(modes map[string]float64, unit string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	supportedUnits := s.Specification().PropertyStrings(PropertySupportedUnits)
	found := utils.SliceContains(unit, supportedUnits)

	if !found {
		return fmt.Errorf("%s: unsupported unit is provided: %s. Supported: %v", s.Name(), unit, supportedUnits)
	}

	for mode := range modes {
		if !utils.SliceContains(mode, s.Specification().PropertyStrings(PropertySupportedModes)) {
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

func (s *service) SetReportingInterval(minutes int) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	// TODO change back to minutes
	if err := s.manager.SetReportingInterval(time.Duration(minutes) * time.Minute); err != nil {
		return fmt.Errorf("failed to set reporting interval: %w", err)
	}

	return nil
}

func (s *service) SendReportingInterval() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	value := s.manager.ReportingInterval()

	// TODO change back to minutes
	message := fimpgo.NewIntMessage(
		EvtConfigIntervalReport,
		s.Name(),
		int64(value.Minutes()),
		nil,
		nil,
		nil,
	)

	err := s.SendMessage(message)
	if err != nil {
		return fmt.Errorf("%s: failed to send reporting interval: %w", s.Name(), err)
	}

	return nil
}
