package virtualmeter

import (
	"fmt"
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/database"
)

const (
	keyDevice = "device"
	dbDomain  = "virtualManager"
)

type (
	Device struct {
		Modes             map[string]float64 `json:"modes"`
		CurrentMode       string             `json:"currentMode"`
		Level             float64            `json:"level"`
		AccumulatedEnergy float64            `json:"accumulatedEnergy"`
		LastTimeUpdated   time.Time          `json:"lastTimeUpdated"`
		Unit              string             `json:"unit"`
		Active            bool               `json:"active"`
	}

	Storage struct {
		db   database.Database
		lock sync.RWMutex
	}
)

func (d *Device) Initialised() bool {
	return d.Modes != nil
}

func NewStorage(db database.Database) *Storage {
	return &Storage{
		db:   database.NewDomainDatabase(dbDomain, db),
		lock: sync.RWMutex{},
	}
}

func (s *Storage) SetDevice(addr string, d *Device) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if err := s.db.Set(keyDevice, addr, d); err != nil {
		return fmt.Errorf("storage: failed to save device by addr - %s: %w", addr, err)
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
		LastTimeUpdated: time.Now(),
	}

	if err := s.db.Set(keyDevice, addr, device); err != nil {
		return fmt.Errorf("storage: failed to mark as deleted by addr - %s: %w", addr, err)
	}

	return nil
}

func (s *Storage) Device(addr string) (*Device, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	device := &Device{}

	ok, err := s.db.Get(keyDevice, addr, device)
	if err != nil {
		return nil, fmt.Errorf("storage: failed to get %s from the database: %w", "device", err)
	} else if !ok {
		return nil, nil
	}

	return device, nil
}
