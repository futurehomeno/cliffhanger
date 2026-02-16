package virtualmeter

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/database"
)

type (
	Manager interface {
		// RegisterThing creates a virtual meter and numeric meter services for a thing based on the existing
		// services. VMS is then added to a think and numeric is added based on whether the virtual meter is already active.
		RegisterThing(thing adapter.Thing, publisher adapter.Publisher) error
		// WithAdapter adds a provided adapter to the provided virtual meter manager. Used to avoid circular dependencies.
		WithAdapter(ad adapter.Adapter)
	}

	manager struct {
		lock                      sync.RWMutex
		ad                        adapter.Adapter
		virtualServices           map[string]adapter.Service
		storage                   *Storage
		energyRecalculationPeriod time.Duration
		garbageCleaningPeriod     time.Duration
	}
)

// NewManager creates a new wrapper with a virtual meter manager.
func NewManager(db database.Database, recalculationPeriod, garbageCleaningPeriod time.Duration) Manager {
	return &manager{
		lock:                      sync.RWMutex{},
		virtualServices:           make(map[string]adapter.Service),
		storage:                   NewStorage(db),
		energyRecalculationPeriod: recalculationPeriod,
		garbageCleaningPeriod:     garbageCleaningPeriod,
	}
}

// WithAdapter adds a provided adapter to the provided virtual meter manager. Used to avoid circular dependencies.
func (m *manager) WithAdapter(ad adapter.Adapter) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.ad = ad
}

// RegisterThing creates a virtual meter and numeric meter services for a thing based on the existing
// services. VMS is then added to a think and numeric is added based on whether the virtual meter is already active.
func (m *manager) RegisterThing(thing adapter.Thing, publisher adapter.Publisher) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, group := range thing.InclusionReport().Groups {
		vmsSpec, numericSpec := m.createVirtualServicesForThing(thing, group)

		if vmsSpec == nil || numericSpec == nil {
			continue
		}

		log.Debugf("[cliff] Register services %s and %s for group %s", vmsSpec.Name, numericSpec.Name, group)

		if err := m.registerVirtualServices(thing, publisher, vmsSpec, numericSpec); err != nil {
			return err
		}
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
	if len(oldModes) > 0 {
		if _, err := m.recalculateEnergy(true, device); err != nil {
			return fmt.Errorf("manager: failed to update energy: %w", err)
		}
	}

	device.Modes = modes
	device.Unit = unit
	device.LastTimeUpdated = time.Now() // this guarantees that event if accumulated energy is not recalculated, the time is updated.

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
	}

	return nil
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

	if err := m.storage.CleanDevice(topic); err != nil {
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

	if device.CurrentMode == newMode && device.Level == newLevel {
		return nil
	}

	if _, err := m.recalculateEnergy(true, device); err != nil {
		return fmt.Errorf("manager: failed to update energy by topic %s: %w", topic, err)
	}

	log.Debugf("[cliff] Update VM with mode=%s level=%v", newMode, newLevel)

	device.Level = newLevel
	device.CurrentMode = newMode

	if err := m.storage.SetDevice(topic, device); err != nil {
		return fmt.Errorf("manager: failed to update device when state changed by topic %s: %w", topic, err)
	}

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

// reset resets accumulated energy for a device found by topic.
func (m *manager) reset(topic string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	device, err := m.storage.Device(topic)
	if err != nil || device == nil {
		return fmt.Errorf("manager: virtual meter reset failed, device - %v: %w", device, err)
	}

	device.AccumulatedEnergy = 0
	device.LastTimeUpdated = time.Now()

	if err := m.storage.SetDevice(topic, device); err != nil {
		return fmt.Errorf("manager: failed to update device when reset by topic %s: %w", topic, err)
	}

	return nil
}

// deleteDeviceEntry removes the device data from the database.
func (m *manager) deleteDeviceEntry(topic string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	device, err := m.storage.Device(topic)
	if err != nil || device == nil {
		return fmt.Errorf("manager: failed to get device by address %s: %w", topic, err)
	}

	if time.Since(device.LastTimeUpdated) > m.garbageCleaningPeriod {
		return m.storage.DeleteDevice(topic)
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

		if !force && time.Since(d.LastTimeUpdated) < m.energyRecalculationPeriod {
			return false, nil
		}

		timeSinceUpdated := time.Since(d.LastTimeUpdated)

		if 2*m.energyRecalculationPeriod < timeSinceUpdated {
			log.Warnf("[cliff] Recalculate energy after a long interruption. Accounting for 2 periods only\nRecalculation period=%v elapsed=%v",
				m.energyRecalculationPeriod, timeSinceUpdated)
		}

		timeSinceUpdatedHours := math.Min(timeSinceUpdated.Hours(), 2*m.energyRecalculationPeriod.Hours())

		increase := timeSinceUpdatedHours * d.Modes[d.CurrentMode] * d.Level
		increase /= 1000

		prevEnergy := d.AccumulatedEnergy

		d.AccumulatedEnergy += increase
		d.LastTimeUpdated = time.Now()

		log.Debugf("[cliff] Update VM energy %v + %v = %v modes=%v active=%t",
			prevEnergy, increase, d.AccumulatedEnergy, d.Modes, d.Active)

		return true, nil
	}

	return false, fmt.Errorf("manager: trying to recalculate energy for 'nil' device")
}

// vmsAddressFromTopic searches for a virtual meter elec services in thing and validates the service address against
// the service address of the incoming topic.
func (m *manager) vmsAddressFromTopic(topic string) (string, error) {
	inAddr, err := fimpgo.NewAddressFromString(topic)
	if err != nil {
		return "", fmt.Errorf("manager: failed to find vms by topic, can't parse in topic: %w", err)
	}

	t := m.ad.ThingByTopic(topic)
	if t == nil {
		return "", fmt.Errorf("manager: failed to find thing for topic %s", topic)
	}

	services := t.Services(VirtualMeterElec)
	if len(services) == 0 {
		return "", fmt.Errorf("manager: failed to find virtual meter service for topic %s", topic)
	}

	for _, s := range services {
		srvAddr, err := fimpgo.NewAddressFromString(s.Topic())
		if err != nil {
			return "", fmt.Errorf("manager: failed to find vms by topic, can't parse out topic: %w", err)
		}

		if srvAddr.ServiceAddress == inAddr.ServiceAddress {
			return s.Topic(), nil
		}
	}

	return "", fmt.Errorf("manager: no vms service found using topic: %s", topic)
}

func (m *manager) normalizeOutLvlSwitchLevel(level int, serviceAddr string) (float64, error) {
	t := m.ad.ThingByTopic(serviceAddr)

	if t == nil {
		return 0.0, fmt.Errorf("manager: failed to find thing for service %s", serviceAddr)
	}

	for _, s := range t.Services(outlvlswitch.OutLvlSwitch) {
		if s.Topic() == serviceAddr {
			maxLevel, _ := s.Specification().PropertyInteger(outlvlswitch.PropertyMaxLvl)
			if maxLevel == 0 {
				return 0.0, fmt.Errorf("manager: max level is set to zero for service %s", serviceAddr)
			}

			return float64(level) / float64(maxLevel), nil
		}
	}

	return 0.0, fmt.Errorf("manager: failed to find outlvlswitch service for thing %s", t.Address())
}

// createVirtualServicesForThing creates a virtual meter and numeric meter services' specifications depending on
// presence of other services. Currently virtual metering is support for following services:
// - outlvlswitch.Service.
func (m *manager) createVirtualServicesForThing(t adapter.Thing, group string) (outLvlSwitchService *fimptype.Service, numericMeterService *fimptype.Service) {
	for _, s := range t.Services("") {
		if len(s.Specification().Groups) == 0 || s.Specification().Groups[0] != group {
			continue
		}

		addr, err := fimpgo.NewAddressFromString(s.Topic())
		if err != nil {
			log.WithError(err).Errorf("manager: failed to parse address from topic %s when creating virtual service", s.Topic())

			continue
		}

		switch s.(type) {
		case outlvlswitch.Service:
			return Specification(
					m.ad.Name(),
					m.ad.Address(),
					addr.ServiceAddress,
					[]string{group},
					[]numericmeter.Unit{numericmeter.UnitW},
					[]string{ModeOn, ModeOff},
				),
				numericmeter.Specification(
					numericmeter.MeterElec,
					m.ad.Name(),
					m.ad.Address(),
					addr.ServiceAddress,
					[]string{group},
					[]numericmeter.Unit{numericmeter.UnitW, numericmeter.UnitKWh},
					numericmeter.WithIsVirtual(),
				)
		default:
			continue
		}
	}

	return nil, nil
}

// registerVirtualServices registers a virtual meter and numeric meter services for a provided thing.
// Avoids updating if virtual service already exist.
// VMS is always added while numeric meter is added only if virtual service is active.
//
//nolint:funlen
func (m *manager) registerVirtualServices(
	thing adapter.Thing,
	publisher adapter.Publisher,
	vmsSpec *fimptype.Service,
	numericSpec *fimptype.Service,
) error {
	// avoid adding virtual service that already exists.
	for _, s := range thing.Services(VirtualMeterElec) {
		if s.Topic() == vmsSpec.Address {
			return nil
		}
	}

	vms := NewService(publisher, &Config{
		Specification: vmsSpec,
		Manager:       m,
	})

	topic := vms.Topic()

	if err := thing.Update(adapter.ThingUpdateAddService(vms)); err != nil {
		return fmt.Errorf("manager: failed to update thing when registering, topic - %s: %w", topic, err)
	}

	srv := numericmeter.NewService(publisher, &numericmeter.Config{
		Specification:     numericSpec,
		Reporter:          newController(topic, m),
		ReportingStrategy: nil,
	})

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
		LastTimeUpdated: time.Now(),
	}

	m.virtualServices[topic] = srv
	if err := m.storage.SetDevice(topic, newDevice); err != nil {
		m.virtualServices[topic] = nil

		return fmt.Errorf("manager: failed to register a device: %w", err)
	}

	return nil
}
