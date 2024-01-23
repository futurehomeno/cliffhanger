package virtualmeter

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/database"
)

type (
	ManagerWrapper interface {
		// RegisterThing creates a virtual meter and numeric meter services for a thing based on the existing
		// services. VMS is then added to a think and numeric is added based on whether the virtual meter is already active.
		RegisterThing(thing adapter.Thing, publisher adapter.Publisher) error
		// WithAdapter adds a provided adapter to the provided virtual meter manager. Used to avoid circular dependencies.
		WithAdapter(ad adapter.Adapter)
		// Manager return an actual virtual meter manager.
		Manager() *manager
	}

	managerWrapper struct {
		*manager
	}

	manager struct {
		lock                      sync.RWMutex
		ad                        adapter.Adapter
		virtualServices           map[string]adapter.Service
		requiredUpdates           map[string]bool
		storage                   *Storage
		energyRecalculationPeriod time.Duration
	}
)

var _ ManagerWrapper = &managerWrapper{}

// NewManagerWrapper creates a new wrapper with a virtual meter manager.
func NewManagerWrapper(db database.Database, recalculationPeriod time.Duration) ManagerWrapper {
	return &managerWrapper{
		manager: &manager{
			lock:                      sync.RWMutex{},
			virtualServices:           make(map[string]adapter.Service),
			requiredUpdates:           make(map[string]bool),
			storage:                   NewStorage(db),
			energyRecalculationPeriod: recalculationPeriod,
		},
	}
}

// Manager return an actual virtual meter manager.
func (m *managerWrapper) Manager() *manager {
	return m.manager
}

// WithAdapter adds a provided adapter to the provided virtual meter manager. Used to avoid circular dependencies.
func (m *managerWrapper) WithAdapter(ad adapter.Adapter) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.ad = ad
}

// RegisterThing creates a virtual meter and numeric meter services for a thing based on the existing
// services. VMS is then added to a think and numeric is added based on whether the virtual meter is already active.
func (m *managerWrapper) RegisterThing(thing adapter.Thing, publisher adapter.Publisher) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	vmsSpec, numericSpec := m.createVirtualServicesForThing(thing)

	if vmsSpec == nil || numericSpec == nil {
		return fmt.Errorf("manager: failed to create virtual meter service for thing %s", thing.Address())
	}

	vms := NewService(publisher, &Config{
		Specification:  vmsSpec,
		ManagerWrapper: m,
	})

	topic := vms.Topic()

	if err := thing.Update(adapter.ThingUpdateAddService(vms)); err != nil {
		return fmt.Errorf("manager: failed to update thing when registering, topic - %s: %w", topic, err)
	}

	srv := numericmeter.NewService(publisher, &numericmeter.Config{
		Specification:     numericSpec,
		Reporter:          newController(topic, m.Manager()),
		ReportingStrategy: nil,
	})

	log.Infof("manager: registering a service template, topic: %s", topic)

	device, err := m.storage.Device(topic)

	if err != nil {
		return fmt.Errorf("manager: failed to get device by address %s: %w", topic, err)
	} else if device != nil && len(device.Modes) != 0 {
		m.virtualServices[topic] = srv

		if err := thing.Update(adapter.ThingUpdateAddService(srv)); err != nil {
			return fmt.Errorf("manager: failed to update thing when registering, topic - %s: %w", topic, err)
		}

		return nil
	}

	newDevice := &Device{
		Modes:           nil,
		Active:          false,
		LastTimeUpdated: time.Now().Format(time.RFC3339),
	}

	m.virtualServices[topic] = srv
	if err := m.storage.SetDevice(topic, newDevice); err != nil {
		m.virtualServices[topic] = nil

		return fmt.Errorf("manager: failed to register a device: %w", err)
	}

	return nil
}

// add adds a virtual service to a device by provided virtual meter topic.
// Updates a thing with the adjusted list of services if the service isn't already added.
func (m *manager) add(topic string, modes map[string]float64, unit string) error { //nolint:cyclop
	m.lock.Lock()
	defer m.lock.Unlock()

	s := m.virtualServices[topic]
	if s == nil {
		return fmt.Errorf("manager: failed to add meter to the thing: %s. no service template found. %v", topic, m.virtualServices)
	}

	thing := m.ad.ThingByTopic(topic)
	if thing == nil {
		return fmt.Errorf("manager: no thing found by topic: %s. can't add meter", topic)
	}

	device, err := m.storage.Device(topic)
	if err != nil || device == nil {
		return fmt.Errorf("manager: failed to get device - %v: %w", device, err)
	}

	oldModes := device.Modes
	if oldModes != nil {
		if _, err := m.recalculateEnergy(true, device); err != nil {
			return fmt.Errorf("manager: failed to update energy: %w", err)
		}
	}

	device.Modes = modes
	device.Unit = unit

	if err := m.storage.SetDevice(topic, device); err != nil {
		return fmt.Errorf("manager: failed add meter, can't save data: %w", err)
	}

	// update thing only if a service has been just added.
	if len(oldModes) == 0 {
		if err := thing.Update(adapter.ThingUpdateAddService(s)); err != nil {
			return fmt.Errorf("manager: failed to update thing. Can't add service. Topic - %s. %w", topic, err)
		}

		if _, err := thing.SendInclusionReport(true); err != nil {
			return fmt.Errorf("manager: failed to send inclusion report on add: %w", err)
		}

		// Marking a device as required to be updated for the first time.
		m.requiredUpdates[topic] = true
	}

	return nil
}

// updateRequired validates if a service by the topic should be updated.
func (m *manager) updateRequired(topic string) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.requiredUpdates[topic]
}

// remove removes a virtual service from a device by provided topic.
// Updates a thing with the adjusted list of services.
func (m *manager) remove(topic string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	s := m.virtualServices[topic]
	if s == nil {
		return fmt.Errorf("manager: failed to remove meter from a thing: %s. No service template found", topic)
	}

	thing := m.ad.ThingByTopic(topic)
	if thing == nil {
		return fmt.Errorf("manager: no thing found by topic: %s. can't remove meter", topic)
	}

	if err := m.storage.DeleteDevice(topic); err != nil {
		return fmt.Errorf("manager: failed to delete meter, can't remove from storage: %w", err)
	}

	if err := thing.Update(adapter.ThingUpdateRemoveService(s)); err != nil {
		return fmt.Errorf("manager: failed to update thing. Can't remove service. Topic: %s. %w", topic, err)
	}

	if _, err := thing.SendInclusionReport(true); err != nil {
		return fmt.Errorf("manager: failed to send inclusion report on remove: %w", err)
	}

	return nil
}

// modes returns a map of modes for a device by provided topic.
func (m *manager) modes(topic string) (map[string]float64, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	device, err := m.storage.Device(topic)
	if err != nil || device == nil {
		return nil, fmt.Errorf("manager: failed to get modes, device - %v: %w", device, err)
	}

	return device.Modes, nil
}

// update updates a device by provided topic with a new mode and level. Recalculates accumulated energy.
// Does nothing if both mode and level hasn't changed.
func (m *manager) update(topic, newMode string, newLevel float64) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	device, err := m.storage.Device(topic)
	if err != nil || device == nil {
		return fmt.Errorf("manager: virtual meter update failed, device - %v: %w", device, err)
	}

	if !device.Initialised() {
		return nil
	}

	if _, err := m.recalculateEnergy(true, device); err != nil {
		return fmt.Errorf("manager: failed to update energy by topic %s: %w", topic, err)
	}

	log.Infof("Updating with the following values: mode %s, level %v", newMode, newLevel)

	device.Level = newLevel
	device.CurrentMode = newMode

	if err := m.storage.SetDevice(topic, device); err != nil {
		return fmt.Errorf("manager: failed to update device when state changed by topic %s: %w", topic, err)
	}

	delete(m.requiredUpdates, topic)

	return nil
}

// report returns an energy report based on provided topic and unit.
func (m *manager) report(topic string, unit numericmeter.Unit) (float64, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	device, err := m.storage.Device(topic)
	if err != nil || device == nil {
		return 0, fmt.Errorf("manager: virtual meter report failed, device - %v: %w", device, err)
	}

	if updated, err := m.recalculateEnergy(false, device); err != nil {
		return 0, fmt.Errorf("manager: failed to update energy by topic %s: %w", topic, err)
	} else if !updated {
		return m.reportPerUnit(device, unit)
	}

	if err := m.storage.SetDevice(topic, device); err != nil {
		return 0, fmt.Errorf("manager: failed to update device when reporting by topic %s: %w", topic, err)
	}

	return m.reportPerUnit(device, unit)
}

func (m *manager) reportPerUnit(device *Device, unit numericmeter.Unit) (float64, error) {
	result := float64(0)

	switch unit { //nolint:exhaustive
	case numericmeter.UnitW:
		result = device.Modes[device.CurrentMode] * device.Level
	case numericmeter.UnitKWh:
		result = device.AccumulatedEnergy
	default:
		return result, fmt.Errorf("manager: report for unknown unit requested: %s", unit)
	}

	return result, nil
}

func (m *manager) reset(topic string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	device, err := m.storage.Device(topic)
	if err != nil || device == nil {
		return fmt.Errorf("manager: virtual meter reset failed, device - %v: %w", device, err)
	}

	device.AccumulatedEnergy = 0
	device.LastTimeUpdated = time.Now().Format(time.RFC3339)

	if err := m.storage.SetDevice(topic, device); err != nil {
		return fmt.Errorf("manager: failed to update device when reset by topic %s: %w", topic, err)
	}

	return nil
}

// updateDeviceActivity updates a device activity for each virtual service of a thing by provided thing address.
func (m *manager) updateDeviceActivity(thingAddr string, active bool) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	thing := m.ad.ThingByAddress(thingAddr)
	if thing == nil {
		return fmt.Errorf("manager: no thing found by address: %s. can't update device activity", thingAddr)
	}

	for _, s := range thing.Services("") {
		topic := s.Topic()
		if m.virtualServices[topic] == nil {
			continue
		}

		device, err := m.storage.Device(topic)
		if err != nil || device == nil {
			return fmt.Errorf("manager: failed to get device - %v: %w", device, err)
		}

		if device.Active != active {
			device.Active = active

			if err := m.storage.SetDevice(topic, device); err != nil {
				return fmt.Errorf("manager: failed to update device when activity changed by address %s: %w", topic, err)
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
func (m *manager) recalculateEnergy(force bool, d *Device) (bool, error) {
	if d != nil {
		if !d.Active {
			return false, nil
		}

		lastUpdated, err := time.Parse(time.RFC3339, d.LastTimeUpdated)
		if err != nil {
			return false, fmt.Errorf("manager: can't parse lastUpdated time (%s): %w", d.LastTimeUpdated, err)
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

	return false, fmt.Errorf("manager: trying to recalculate energy for 'nil' device")
}

func (m *manager) vmsAddressFromTopic(topic string) (string, error) {
	t := m.ad.ThingByTopic(topic)
	if t == nil {
		return "", fmt.Errorf("manager: failed to find thing for topic %s", topic)
	}

	s := t.Services(VirtualMeterElec)
	if len(s) == 0 {
		return "", fmt.Errorf("manager: failed to find virtual meter service for topic %s", topic)
	}

	return s[0].Topic(), nil
}

func (m *manager) normalizeOutLvlSwitchLevel(level int64, serviceAddr string) float64 {
	t := m.ad.ThingByTopic(serviceAddr)

	if t == nil {
		log.Errorf("manager: failed to find thing for service %s", serviceAddr)

		return 0.0
	}

	for _, s := range t.Services(outlvlswitch.OutLvlSwitch) {
		if s.Topic() == serviceAddr {
			maxLevel, _ := s.Specification().PropertyInteger(outlvlswitch.PropertyMaxLvl)
			if maxLevel == 0 {
				log.Errorf("manager: max level is set to zero for service %s", serviceAddr)

				return 0.0
			}

			return float64(level) / float64(maxLevel)
		}
	}

	log.Errorf("manager: failed to find outlvlswitch service for thing %s", t.Address())

	return 0.0
}

// createVirtualServicesForThing creates a virtual meter and numeric meter services' specifications depending on
// presence of other services. Currently virtual metering is support for following services:
// - outlvlswitch.Service.
func (m *manager) createVirtualServicesForThing(t adapter.Thing) (*fimptype.Service, *fimptype.Service) {
	for _, s := range t.Services("") {
		switch s.(type) {
		case outlvlswitch.Service:
			return Specification(
					m.ad.Name(),
					m.ad.Address(),
					t.Address(),
					t.InclusionReport().Groups,
					[]numericmeter.Unit{numericmeter.UnitW, numericmeter.UnitKWh},
					[]string{ModeOn, ModeOff},
				),
				numericmeter.Specification(
					numericmeter.MeterElec,
					m.ad.Name(),
					m.ad.Address(),
					t.Address(),
					t.InclusionReport().Groups,
					[]numericmeter.Unit{numericmeter.UnitW, numericmeter.UnitKWh},
					numericmeter.WithIsVirtual(),
				)
		default:
			continue
		}
	}

	return nil, nil
}
