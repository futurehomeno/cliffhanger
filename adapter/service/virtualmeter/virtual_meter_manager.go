package virtualmeter

import (
	"errors"
	"fmt"
	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/task"
	log "github.com/sirupsen/logrus"
	"math"
	"strings"
	"sync"
	"time"
)

type (
	VirtualMeterManager interface {
		Add(addr string, modes map[string]float64, unit string) error
		Remove(addr string) error
		Modes(addr string) (map[string]float64, error)
		SetReportingInterval(duration time.Duration) error
		ReportingInterval() time.Duration

		// RegisterDevice saves service by address that will be added to the thing on respective fimp message or
		// add service to the thing immediately if it was initialised previously.
		RegisterDevice(thing adapter.Thing, addr string, service adapter.Service) error
		// Update updates a virtual meter for a device by a given addr with a new mode and level.
		Update(addr, mode string, level float64) error
		// Report returns a value to report based on a provided unit.
		Report(addr, unit string) (float64, error)
	}

	virtualMeterManager struct {
		lock             sync.RWMutex
		ad               adapter.Adapter
		serviceTemplates map[string]adapter.Service
		storage          Storage
		taskManager      task.Manager
	}
)

var (
	_ VirtualMeterManager = &virtualMeterManager{}
)

// NewVirtualMeterManager creates a new virtual meter manager with basic initialisation.
func NewVirtualMeterManager(workdir string) VirtualMeterManager {
	return &virtualMeterManager{
		lock:             sync.RWMutex{},
		serviceTemplates: make(map[string]adapter.Service),
		storage:          NewStorage(workdir),
	}
}

// WithAdapter adds a provided adapter to the provided virtual meter manager. Used to avoid circular dependencies.
func WithAdapter(meter VirtualMeterManager, ad adapter.Adapter) VirtualMeterManager {
	vmeter, ok := meter.(*virtualMeterManager)
	if !ok {
		log.Fatal("failed to inject adapter into virtual meter")
	}

	vmeter.ad = ad

	return vmeter
}

// WithTaskManager adds a provided task manager to the provided virtual meter manager. Used to avoid circular dependencies.
func WithTaskManager(meter VirtualMeterManager, manager task.Manager) VirtualMeterManager {
	vmeter, ok := meter.(*virtualMeterManager)
	if !ok {
		log.Fatal("failed to inject adapter into virtual meter")
	}

	vmeter.taskManager = manager

	return vmeter
}

func (m *virtualMeterManager) Add(addr string, modes map[string]float64, unit string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	addr = m.thingAddrFromTopic(addr)

	log.Infof("Virtual meter: adding meter")

	s := m.serviceTemplates[addr]
	if s == nil {
		return fmt.Errorf("failed to add meter to the thing: %s. no service template found. %v", addr, m.serviceTemplates)
	}

	thing := m.ad.ThingByAddress(addr)
	if thing == nil {
		return fmt.Errorf("no thing found by address: %s. can't add meter", addr)
	}

	device, err := m.storage.Device(addr)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	oldModes := device.Modes
	if oldModes != nil {
		if err := m.recalculateEnergy(&device); err != nil {
			return fmt.Errorf("failed to update energy: %w", err)
		}
	}

	device.Modes = modes
	device.Unit = unit

	if err := m.storage.SetDeviceEntry(addr, device); err != nil {
		return fmt.Errorf("failed add meter, can't save data: %w", err)
	}

	// Update thing only if a service has been just added.
	if oldModes == nil {
		if err := thing.Update(true, adapter.ThingUpdateAddService(s)); err != nil {
			return fmt.Errorf("failed to update thing. Can't add service. Addr: %s. %w", addr, err)
		}
	}

	return nil
}

func (m *virtualMeterManager) Remove(addr string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	addr = m.thingAddrFromTopic(addr)

	s := m.serviceTemplates[addr]
	if s == nil {
		return fmt.Errorf("failed to remove meter from a thing: %s. No service template found", addr)
	}

	thing := m.ad.ThingByAddress(addr)
	if thing == nil {
		return fmt.Errorf("no thing found by address: %s. can't remove meter", addr)
	}

	if err := m.storage.DeleteDevice(addr); err != nil {
		return fmt.Errorf("failed to delete meter, can't remove from storage: %w", err)
	}

	if err := thing.Update(true, adapter.ThingUpdateRemoveService(m.serviceTemplates[addr])); err != nil {
		return fmt.Errorf("failed to update thing. Can't remove service. Addr: %s. %w", addr, err)
	}

	return nil
}

func (m *virtualMeterManager) Modes(addr string) (map[string]float64, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	addr = m.thingAddrFromTopic(addr)

	device, err := m.storage.Device(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to get modes: %w", err)
	}

	return device.Modes, nil
}

func (m *virtualMeterManager) SetReportingInterval(duration time.Duration) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.taskManager == nil {
		return fmt.Errorf("virtual meter does't have task manager initialised, can't update task interval")
	}

	if err := m.storage.SetReportingInterval(duration); err != nil {
		return fmt.Errorf("failed to save repoting interval: %w", err)
	}

	if err := m.taskManager.UpdateTaskInterval(TaskVirtualReporter, duration); err != nil {
		return fmt.Errorf("failed to update a task with new reporting interval: %w", err)
	}

	return nil
}

func (m *virtualMeterManager) ReportingInterval() time.Duration {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.storage.ReportingInterval()
}

func (m *virtualMeterManager) RegisterDevice(thing adapter.Thing, addr string, service adapter.Service) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	addr = m.thingAddrFromTopic(addr)

	log.Infof("Virtual meter: registering a service template, addr: %s", addr)

	device, err := m.storage.Device(addr)
	if err != nil && !errors.As(err, &ErrorEntryNotFound{}) {

		return fmt.Errorf("virtual meter: failed to get device by addr %s: %w", addr, err)
	} else if err == nil && device.Modes != nil {
		m.serviceTemplates[addr] = service

		if err := thing.Update(false, adapter.ThingUpdateAddService(service)); err != nil {
			return fmt.Errorf("failed to update thing when registering, addr %s, error %w", addr, err)
		}

		return nil
	}

	deviceEntry := DeviceEntry{
		Modes:           nil,
		Active:          false,
		LastTimeUpdated: time.Now().Format(time.RFC3339),
	}

	m.serviceTemplates[addr] = service
	if err := m.storage.SetDeviceEntry(addr, deviceEntry); err != nil {
		m.serviceTemplates[addr] = nil

		return fmt.Errorf("failed to register a device, database error: %w", err)
	}

	return nil
}

func (m *virtualMeterManager) Update(addr, newMode string, newLevel float64) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	addr = m.thingAddrFromTopic(addr)

	device, err := m.storage.Device(addr)
	if err != nil {
		return fmt.Errorf("virtual meter update failed: %w", err)
	}

	if !device.Initialised() {
		return nil
	}

	if device.CurrentMode == newMode && device.Level == newLevel {
		return nil
	}

	if err := m.recalculateEnergy(&device); err != nil {
		return fmt.Errorf("failed to update energy by addr %s: %w", addr, err)
	}

	log.Infof("Updating with the following values: mode %s, level %v", newMode, newLevel)

	device.Level = newLevel
	device.CurrentMode = newMode

	if err := m.storage.SetDeviceEntry(addr, device); err != nil {
		return fmt.Errorf("failed to update device when state changed by addr %s : %w", addr, err)
	}

	return nil
}

func (m *virtualMeterManager) Report(addr, unit string) (float64, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	addr = m.thingAddrFromTopic(addr)

	device, err := m.storage.Device(addr)
	if err != nil {
		return 0, fmt.Errorf("virtual meter report failed: %w", err)
	}

	if err := m.recalculateEnergy(&device); err != nil {
		return 0, fmt.Errorf("failed to update energy by addr %s: %w", addr, err)
	}

	if err := m.storage.SetDeviceEntry(addr, device); err != nil {
		return 0, fmt.Errorf("failed to update device when reporting by addr %s : %w", addr, err)
	}

	result := float64(0)

	switch unit {
	case "W":
		result = 12 //device.Modes[device.CurrentMode] * device.Level
	case "kWh":
		result = device.AccumulatedEnergy
	default:
		return 0, fmt.Errorf("virtual meter: report for unkonwn unit requested: %s", unit)
	}

	return result, nil
}

// RecalculateEnergy calculates the energy consumptions since the last time measures and adds to total.
// If the time since last measured is bigger than a reporting interval, we assume that there was a fault in reporting
// and device stayed reporting interval with the latest mode after which we stop measuring.
func (m *virtualMeterManager) recalculateEnergy(d *DeviceEntry) error {
	if d != nil {
		lastUpdated, err := time.Parse(time.RFC3339, d.LastTimeUpdated)
		if err != nil {
			return fmt.Errorf("can't parse lastUpdated time (%s): %w", d.LastTimeUpdated, err)
		}

		reportingInterval := m.storage.ReportingInterval()
		timeSinceUpdated := time.Since(lastUpdated)

		if 2*reportingInterval < timeSinceUpdated {
			log.Warnf("Recalculating enegry after a long interuption. Accounting for reporting interval only." +
				fmt.Sprintf(" \nReporting interval: %v, Time elapsed: %v", reportingInterval, timeSinceUpdated))
		}

		timeSinceUpdatedHours := math.Min(timeSinceUpdated.Hours(), reportingInterval.Hours())

		increase := timeSinceUpdatedHours * d.Modes[d.CurrentMode] * d.Level
		increase /= 1000
		log.Debugf("Updating accumulated energy. Current value: %v, increase: %v, modes: %v, mode: %s", d.AccumulatedEnergy, increase, d.Modes, d.CurrentMode)

		d.AccumulatedEnergy += increase
		d.LastTimeUpdated = time.Now().Format(time.RFC3339)

		return nil
	}

	return nil
}

func (m *virtualMeterManager) thingAddrFromTopic(topic string) string {
	parts := strings.Split(topic, ":")

	return parts[len(parts)-1]
}
