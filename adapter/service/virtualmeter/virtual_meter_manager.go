package virtualmeter

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/database"
)

type (
	Manager interface {
		Add(addr string, modes map[string]float64, unit string) error
		Remove(addr string) error
		Modes(addr string) (map[string]float64, error)

		// RegisterDevice creates and saves a service by address that will be added to the thing on respective fimp message or
		// add service to the thing immediately if it was initialised previously.
		RegisterDevice(thing adapter.Thing, addr string, publisher adapter.Publisher, spec *fimptype.Service) error
		// Update updates a virtual meter for a device by a given addr with a new mode and level.
		Update(topic, mode string, level float64) error
		// UpdateRequired validates if a service by the topic should be updated.
		UpdateRequired(topic string) bool
		// Report returns a value to report based on a provided unit.
		Report(topic string, unit numericmeter.Unit) (float64, error)
		// WithAdapter adds a provided adapter to the provided virtual meter manager. Used to avoid circular dependencies.
		WithAdapter(ad adapter.Adapter)

		updateDeviceActivity(thingAddr string, active bool) error
	}

	virtualMeterManager struct {
		lock                      sync.RWMutex
		ad                        adapter.Adapter
		virtualServices           map[string]adapter.Service
		requiredUpdates           map[string]bool
		storage                   *Storage
		energyRecalculationPeriod time.Duration
	}
)

var _ Manager = &virtualMeterManager{}

// NewVirtualMeterManager creates a new virtual meter manager with basic initialisation.
func NewVirtualMeterManager(db database.Database, recalculationPeriod time.Duration) Manager {
	return &virtualMeterManager{
		lock:                      sync.RWMutex{},
		virtualServices:           make(map[string]adapter.Service),
		requiredUpdates:           make(map[string]bool),
		storage:                   NewStorage(db),
		energyRecalculationPeriod: recalculationPeriod,
	}
}

// WithAdapter adds a provided adapter to the provided virtual meter manager. Used to avoid circular dependencies.
func (m *virtualMeterManager) WithAdapter(ad adapter.Adapter) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.ad = ad
}

// Add adds a virtual service to a device by provided topic.
// Updates a thing with the adjusted list of services if the service isn't already added.
func (m *virtualMeterManager) Add(topic string, modes map[string]float64, unit string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	serviceAddr := m.serviceAddrFromTopic(topic)

	s := m.virtualServices[serviceAddr]
	if s == nil {
		return fmt.Errorf("failed to add meter to the thing: %s. no service template found. %v", serviceAddr, m.virtualServices)
	}

	thing := m.ad.ThingByTopic(topic)
	if thing == nil {
		return fmt.Errorf("no thing found by address: %s. can't add meter", topic)
	}

	device, err := m.storage.Device(serviceAddr)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	oldModes := device.Modes
	if oldModes != nil {
		if _, err := m.recalculateEnergy(true, &device); err != nil {
			return fmt.Errorf("failed to update energy: %w", err)
		}
	}

	device.Modes = modes
	device.Unit = unit

	if err := m.storage.SetDevice(serviceAddr, device); err != nil {
		return fmt.Errorf("failed add meter, can't save data: %w", err)
	}

	// Update thing only if a service has been just added.
	if oldModes == nil {
		if err := thing.Update(true, adapter.ThingUpdateAddService(s)); err != nil {
			return fmt.Errorf("failed to update thing. Can't add service. Topic: %s. %w", topic, err)
		}

		// Marking a device as required to be updated for the first time.
		m.requiredUpdates[serviceAddr] = true
	}

	return nil
}

// UpdateRequired validates if a service by the topic should be updated.
func (m *virtualMeterManager) UpdateRequired(topic string) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()

	serviceAddr := m.serviceAddrFromTopic(topic)

	return m.requiredUpdates[serviceAddr]
}

// Remove removes a virtual service from a device by provided topic.
// Updates a thing with the adjusted list of services.
func (m *virtualMeterManager) Remove(topic string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	serviceAddr := m.serviceAddrFromTopic(topic)

	s := m.virtualServices[serviceAddr]
	if s == nil {
		return fmt.Errorf("failed to remove meter from a thing: %s. No service template found", serviceAddr)
	}

	thing := m.ad.ThingByTopic(topic)
	if thing == nil {
		return fmt.Errorf("no thing found by address: %s. can't remove meter", topic)
	}

	if err := m.storage.DeleteDevice(serviceAddr); err != nil {
		return fmt.Errorf("failed to delete meter, can't remove from storage: %w", err)
	}

	if err := thing.Update(true, adapter.ThingUpdateRemoveService(s)); err != nil {
		return fmt.Errorf("failed to update thing. Can't remove service. Topic: %s. %w", topic, err)
	}

	return nil
}

// Modes returns a map of modes for a device by provided topic.
func (m *virtualMeterManager) Modes(topic string) (map[string]float64, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	serviceAddr := m.serviceAddrFromTopic(topic)

	device, err := m.storage.Device(serviceAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get modes: %w", err)
	}

	return device.Modes, nil
}

// RegisterDevice creates and saves the numericmeter service by the thing address.
// If the meter (not virtual) is already added and can be found in the store the thing is immediately initialised with a service.
func (m *virtualMeterManager) RegisterDevice(thing adapter.Thing, topic string, publisher adapter.Publisher, spec *fimptype.Service) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	serviceAddr := m.serviceAddrFromTopic(topic)

	srv := numericmeter.NewService(publisher, &numericmeter.Config{
		Specification:     spec,
		Reporter:          newController(topic, m),
		ReportingStrategy: nil,
	})

	log.Infof("Virtual meter: registering a service template, topic: %s", topic)

	device, err := m.storage.Device(serviceAddr)
	if err != nil && !errors.As(err, &ErrorEntryNotFound{}) {
		return fmt.Errorf("virtual meter: failed to get device by address %s: %w", serviceAddr, err)
	} else if err == nil && device.Modes != nil {
		m.virtualServices[serviceAddr] = srv

		if err := thing.Update(false, adapter.ThingUpdateAddService(srv)); err != nil {
			return fmt.Errorf("failed to update thing when registering, topic %s, error %w", topic, err)
		}

		return nil
	}

	newDevice := Device{
		Modes:           nil,
		Active:          false,
		LastTimeUpdated: time.Now().Format(time.RFC3339),
	}

	m.virtualServices[serviceAddr] = srv
	if err := m.storage.SetDevice(serviceAddr, newDevice); err != nil {
		m.virtualServices[serviceAddr] = nil

		return fmt.Errorf("failed to register a device, database error: %w", err)
	}

	return nil
}

// Update updates a device by provided topic with a new mode and level. Recalculates accumulated energy.
// Does nothing if both mode and level hasn't changed.
func (m *virtualMeterManager) Update(topic, newMode string, newLevel float64) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	serviceAddr := m.serviceAddrFromTopic(topic)

	device, err := m.storage.Device(serviceAddr)
	if err != nil {
		return fmt.Errorf("virtual meter update failed: %w", err)
	}

	if !device.Initialised() {
		return nil
	}

	if _, err := m.recalculateEnergy(true, &device); err != nil {
		return fmt.Errorf("failed to update energy by topic %s: %w", topic, err)
	}

	log.Infof("Updating with the following values: mode %s, level %v", newMode, newLevel)

	device.Level = newLevel
	device.CurrentMode = newMode

	if err := m.storage.SetDevice(serviceAddr, device); err != nil {
		return fmt.Errorf("failed to update device when state changed by address %s : %w", serviceAddr, err)
	}

	delete(m.requiredUpdates, serviceAddr)

	return nil
}

// Report returns an energy report based on provided topic and unit.
func (m *virtualMeterManager) Report(topic string, unit numericmeter.Unit) (float64, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	serviceAddr := m.serviceAddrFromTopic(topic)

	device, err := m.storage.Device(serviceAddr)
	if err != nil {
		return 0, fmt.Errorf("virtual meter report failed: %w", err)
	}

	if updated, err := m.recalculateEnergy(false, &device); err != nil {
		return 0, fmt.Errorf("failed to update energy by topic %s: %w", topic, err)
	} else if !updated {
		return device.AccumulatedEnergy, nil
	}

	if err := m.storage.SetDevice(serviceAddr, device); err != nil {
		return 0, fmt.Errorf("failed to update device when reporting by address %s : %w", serviceAddr, err)
	}

	result := float64(0)

	switch unit { //nolint:exhaustive
	case numericmeter.UnitW:
		result = device.Modes[device.CurrentMode] * device.Level
	case numericmeter.UnitKWh:
		result = device.AccumulatedEnergy
	default:
		return result, fmt.Errorf("virtual meter: report for unknown unit requested: %s", unit)
	}

	return result, nil
}

// updateDeviceActivity updates a device activity for each virtual service of a thing by provided thing address.
func (m *virtualMeterManager) updateDeviceActivity(thingAddr string, active bool) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	thing := m.ad.ThingByAddress(thingAddr)
	if thing == nil {
		return fmt.Errorf("no thing found by address: %s. can't update device activity", thingAddr)
	}

	for _, s := range thing.Services("") {
		serviceAddr := m.serviceAddrFromTopic(s.Topic())
		if m.virtualServices[serviceAddr] == nil {
			continue
		}

		device, err := m.storage.Device(serviceAddr)
		if err != nil {
			return fmt.Errorf("failed to get device: %w", err)
		}

		if device.Active != active {
			device.Active = active

			if err := m.storage.SetDevice(serviceAddr, device); err != nil {
				return fmt.Errorf("failed to update device when activity changed by address %s : %w", serviceAddr, err)
			}
		}
	}

	return nil
}

// recalculateEnergy calculates the energy consumptions by applying the following logic:
// - if the time elapsed since last recalculation < recalculationPeriod nothing happens.
// - if forced, the step above is skipped
// - if time elapsed since last recalculation > 2 * recalculationPeriod we consider this unexpected behaviour and
// account for only single recalculationPeriod timeframe with the latest state.
func (m *virtualMeterManager) recalculateEnergy(force bool, d *Device) (bool, error) {
	if d != nil {
		if !d.Active {
			return false, nil
		}

		lastUpdated, err := time.Parse(time.RFC3339, d.LastTimeUpdated)
		if err != nil {
			return false, fmt.Errorf("can't parse lastUpdated time (%s): %w", d.LastTimeUpdated, err)
		}

		if !force && time.Since(lastUpdated) < m.energyRecalculationPeriod {
			return false, nil
		}

		timeSinceUpdated := time.Since(lastUpdated)

		if 2*m.energyRecalculationPeriod < timeSinceUpdated {
			log.Warnf("Recalculating enegry after a long interuption. Accounting for 2 recalculation periods only." +
				fmt.Sprintf(" \nRecalculation period: %v, Time elapsed: %v", m.energyRecalculationPeriod, timeSinceUpdated))
		}

		timeSinceUpdatedHours := math.Min(timeSinceUpdated.Hours(), 2*m.energyRecalculationPeriod.Hours())

		increase := timeSinceUpdatedHours * d.Modes[d.CurrentMode] * d.Level
		increase /= 1000
		log.Debugf("Updating accumulated energy. Current values: %v, increase (Wh): %v, modes: %v, mode: %s",
			d.AccumulatedEnergy, increase*1000, d.Modes, d.CurrentMode)

		d.AccumulatedEnergy += increase
		d.LastTimeUpdated = time.Now().Format(time.RFC3339)

		return true, nil
	}

	return false, fmt.Errorf("trying to recalculate energy for 'nil' device")
}

// serviceAddrFromTopic is required as some entities don't keep a thing addr but do keep a topic.
func (m *virtualMeterManager) serviceAddrFromTopic(topic string) string {
	parts := strings.Split(topic, ":")

	return parts[len(parts)-1]
}
