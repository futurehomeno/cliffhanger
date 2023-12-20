package virtualmeter

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/database"
)

const (
	keyDevice            = "device"
	keyReportingInterval = "reportingInterval"
	dbDomain             = "virtualManager"

	defaultReportingInterval = time.Minute * 30
)

type (
	Device struct {
		Modes             map[string]float64 `json:"modes"`
		CurrentMode       string             `json:"currentMode"`
		Level             float64            `json:"level"`
		AccumulatedEnergy float64            `json:"accumulatedEnergy"`
		LastTimeUpdated   string             `json:"lastTimeUpdated"`
		Unit              string             `json:"unit"`
		Active            bool               `json:"active"`
	}

	Storage struct {
		db   database.Database
		lock sync.RWMutex
	}

	ErrorEntryNotFound struct {
		m string
	}
)

func (e ErrorEntryNotFound) Error() string {
	return e.m
}

func (d *Device) Initialised() bool {
	return d.Modes != nil
}

func NewStorage(db database.Database) *Storage {
	return &Storage{
		db:   database.NewDomainDatabase(dbDomain, db),
		lock: sync.RWMutex{},
	}
}

func (s *Storage) SetDevice(addr string, d Device) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if err := s.db.Set(keyDevice, addr, d); err != nil {
		return fmt.Errorf("failed to save device by addr: %s. %w", addr, err)
	}

	return nil
}

// DeleteDevice marks device as inactive by removing modes.
func (s *Storage) DeleteDevice(addr string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	device := Device{
		Modes:           nil,
		Active:          false,
		LastTimeUpdated: time.Now().Format(time.RFC3339),
	}

	if err := s.db.Set(keyDevice, addr, device); err != nil {
		return fmt.Errorf("failed to mark as deleted by addr %s: %w", addr, err)
	}

	return nil
}

func (s *Storage) Device(addr string) (Device, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	device := Device{}

	ok, err := s.db.Get(keyDevice, addr, &device)
	if err := s.processGetError(addr, "device", ok, err); err != nil {
		return Device{}, err
	}

	return device, nil
}

func (s *Storage) ReportingInterval() time.Duration {
	s.lock.RLock()
	defer s.lock.RUnlock()

	interval := ""

	ok, err := s.db.Get(keyReportingInterval, "", &interval)
	if err := s.processGetError("", "repoting interval", ok, err); err != nil {
		log.WithError(err).Errorf("db: failed to get reporting interval")

		return defaultReportingInterval
	}

	duration, err := time.ParseDuration(interval)
	if err != nil {
		log.WithError(err).Errorf("db: failed to parse interval ad duration, interval: %s", interval)

		return defaultReportingInterval
	}

	return duration
}

func (s *Storage) SetReportingInterval(duration time.Duration) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if err := s.db.Set(keyReportingInterval, "", duration.String()); err != nil {
		return fmt.Errorf("db: failed to save repoting interval: %w", err)
	}

	return nil
}

func (s *Storage) processGetError(addr, fieldName string, ok bool, err error) error {
	if err != nil {
		return fmt.Errorf("failed to get %s from the database: %w", fieldName, err)
	}

	if !ok {
		return ErrorEntryNotFound{m: fmt.Sprintf("no current %s found by addr: %s. ", fieldName, addr)}
	}

	return nil
}
