package virtualmeter //nolint:testpackage

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/database"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedoutlvlswitch "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/outlvlswitch"
)

const (
	addr = "test"
)

var (
	outLvlSwitchService = outlvlswitch.NewService(
		nil,
		&outlvlswitch.Config{
			Specification: &fimptype.Service{
				Name:    outlvlswitch.OutLvlSwitch,
				Address: addr,
			},
		})

	meterElecService = numericmeter.NewService(
		nil,
		&numericmeter.Config{
			Specification: &fimptype.Service{
				Name:    numericmeter.MeterElec,
				Address: addr,
			},
		})

	outLvlSwitchServiceFullAddr = outlvlswitch.NewService(
		nil,
		&outlvlswitch.Config{
			Specification: &fimptype.Service{
				Name:    outlvlswitch.OutLvlSwitch,
				Address: "/rt:dev/rn:test/ad:test/sv:out_level_switch/ad:test_ch1",
				Groups:  []string{"ch1"},
			},
		})
)

// levelSwitchForceReport returns an outlvlswitch.Service mock at topic asserting that
// add() forces an initial level report so a meter added to an already-stable device
// doesn't accrue zero energy forever.
func levelSwitchForceReport(t *testing.T, topic string) outlvlswitch.Service {
	t.Helper()

	m := mockedoutlvlswitch.NewService(t)
	m.EXPECT().Topic().Return(topic)
	m.EXPECT().SendLevelReport(true).Return(true, nil).Once()

	return m
}

func TestVirtualMeterManager_Add(t *testing.T) { //nolint:paralleltest
	cases := []struct {
		name              string
		configuredService adapter.Service
		existingDevice    *Device
		mockedThing       adapter.Thing
		teardown          func(t *testing.T)
		expectError       bool
	}{
		{
			name:              "should fail when no virtual service configured",
			configuredService: nil,
			teardown:          func(t *testing.T) { t.Helper() },
			expectError:       true,
		},
		{
			name:              "should fail when no thing found",
			configuredService: outLvlSwitchService,
			mockedThing:       nil,
			teardown:          func(t *testing.T) { t.Helper() },
			expectError:       true,
		},
		{
			name:              "should fail when no device found",
			configuredService: outLvlSwitchService,
			mockedThing:       mockedadapter.NewThing(t),
			teardown:          func(t *testing.T) { t.Helper() },
			expectError:       true,
		},
		{
			// Regression: Update() inserts the service before the report is published, so a retry
			// of an add whose publish failed finds the service already there. The report must go
			// out anyway, or the meter never reaches the hub while add() keeps answering success.
			name:              "should re-announce an already added service without adding it again",
			configuredService: outLvlSwitchService,
			existingDevice:    &Device{Modes: map[string]float64{"on": 123}, CurrentMode: "on"},
			mockedThing: mockedadapter.NewThing(t).
				WithSendInclusionReport(true, true, true, nil).
				WithServices(numericmeter.MeterElec, true, []adapter.Service{meterElecService}),
			teardown:    adapterhelper.TearDownAdapter(workdir)[0],
			expectError: false,
		},
		{
			// Regression: the same retry also used to skip the forced level report, because the
			// service being present was read as "already seeded". Nothing else ever seeds
			// CurrentMode for a device that is not changing state, so the meter stayed at zero.
			// No outlvlswitch stub on the case above: a device that already has a CurrentMode
			// must not be re-seeded.
			name:              "should force an initial level report when a retried add finds the service already present",
			configuredService: outLvlSwitchService,
			existingDevice:    &Device{Modes: map[string]float64{"on": 123}},
			mockedThing: mockedadapter.NewThing(t).
				WithSendInclusionReport(true, true, true, nil).
				WithServices(numericmeter.MeterElec, true, []adapter.Service{meterElecService}).
				WithServices(outlvlswitch.OutLvlSwitch, true, []adapter.Service{levelSwitchForceReport(t, addr)}),
			teardown:    adapterhelper.TearDownAdapter(workdir)[0],
			expectError: false,
		},
		{
			name:              "should succeed and updated modes, and send inclusion report",
			configuredService: outLvlSwitchService,
			existingDevice:    &Device{Modes: nil},
			mockedThing: mockedadapter.NewThing(t).
				WithUpdate(true, nil).
				WithSendInclusionReport(true, true, true, nil).
				WithServices(numericmeter.MeterElec, true, []adapter.Service{}).
				WithServices(outlvlswitch.OutLvlSwitch, true, []adapter.Service{}),
			teardown:    adapterhelper.TearDownAdapter(workdir)[0],
			expectError: false,
		},
		{
			name:              "should force an initial level report on first add so the meter isn't stuck at zero",
			configuredService: outLvlSwitchService,
			existingDevice:    &Device{Modes: nil},
			mockedThing: mockedadapter.NewThing(t).
				WithUpdate(true, nil).
				WithSendInclusionReport(true, true, true, nil).
				WithServices(numericmeter.MeterElec, true, []adapter.Service{}).
				WithServices(outlvlswitch.OutLvlSwitch, true, []adapter.Service{levelSwitchForceReport(t, addr)}),
			teardown:    adapterhelper.TearDownAdapter(workdir)[0],
			expectError: false,
		},
	}

	for _, cc := range cases { //nolint:paralleltest
		c := cc
		t.Run(c.name, func(t *testing.T) {
			defer c.teardown(t)

			db, _ := database.NewDatabase(workdir)
			mr := NewManager(db, time.Second, time.Hour)
			m := mr.(*manager) //nolint:forcetypeassert

			// Expected in every case: the thing is now resolved before the manager lock is taken,
			// so the lookup happens even when the service template check rejects the call.
			mockAdapter := mockedadapter.NewAdapter(t).WithThingByTopic(addr, true, c.mockedThing)

			if c.existingDevice != nil {
				err := m.storage.SetDevice(addr, c.existingDevice)
				assert.NoError(t, err, "should set device")
			}

			m.ad = mockAdapter
			m.virtualServices = map[string]adapter.Service{addr: c.configuredService}

			modes := map[string]float64{"on": 123}

			err := m.add(addr, modes, "W")
			if c.expectError {
				assert.Error(t, err, "should fail to add a meter")
			} else {
				assert.NoError(t, err, "should add a meter")
				modes, err := m.modes(addr)
				assert.NoError(t, err, "should get modes")
				assert.Equal(t, modes, modes, "should add modes")
			}
		})
	}
}

func TestManager_Remove(t *testing.T) { //nolint:paralleltest
	cases := []struct {
		name              string
		configuredService adapter.Service
		mockedThing       adapter.Thing
		teardown          func(t *testing.T)
		expectError       bool
	}{
		{
			name:              "should fail to remove when no service configured",
			configuredService: nil,
			teardown:          func(t *testing.T) { t.Helper() },
			expectError:       true,
		},
		{
			name:              "should fail to remove when no thing found",
			configuredService: outLvlSwitchService,
			mockedThing:       nil,
			teardown:          func(t *testing.T) { t.Helper() },
			expectError:       true,
		},
		{
			name:              "should fail when thing update failed",
			configuredService: outLvlSwitchService,
			mockedThing:       mockedadapter.NewThing(t).WithUpdate(true, errors.New("some")),
			teardown:          adapterhelper.TearDownAdapter(workdir)[0],
			expectError:       true,
		},
		{
			name:              "should succeed and remove modes",
			configuredService: outLvlSwitchService,
			mockedThing: mockedadapter.NewThing(t).
				WithUpdate(true, nil).
				WithSendInclusionReport(true, true, true, nil),
			teardown:    adapterhelper.TearDownAdapter(workdir)[0],
			expectError: false,
		},
	}

	for _, cc := range cases { //nolint:paralleltest
		c := cc
		t.Run(c.name, func(t *testing.T) {
			defer c.teardown(t)

			db, _ := database.NewDatabase(workdir)
			mr := NewManager(db, time.Second, time.Hour)
			m := mr.(*manager) //nolint:forcetypeassert

			// Expected in every case: the thing is now resolved before the manager lock is taken,
			// so the lookup happens even when the service template check rejects the call.
			mockAdapter := mockedadapter.NewAdapter(t).WithThingByTopic(addr, true, c.mockedThing)

			err := m.storage.SetDevice(addr, &Device{Modes: make(map[string]float64)})
			assert.NoError(t, err, "should set device")

			m.ad = mockAdapter
			m.virtualServices = map[string]adapter.Service{addr: c.configuredService}

			err = m.remove(addr)
			if c.expectError {
				assert.Error(t, err, "should fail to add a meter")
			} else {
				assert.NoError(t, err, "should add a meter")
				modes, err := m.modes(addr)
				assert.NoError(t, err, "should get modes")
				assert.Equal(t, map[string]float64(nil), modes, "should remove modes")
			}
		})
	}
}

// TestManager_QueriesAdapterWithoutHoldingLock pins the lock order. Thing factories call
// RegisterThing while the adapter lock is held, so an adapter query made from under the manager
// lock inverts that order and deadlocks - a failure that only shows up as a hung process.
func TestManager_QueriesAdapterWithoutHoldingLock(t *testing.T) { //nolint:paralleltest
	workdir := t.TempDir()

	db, err := database.NewDatabase(workdir)
	assert.NoError(t, err, "should create database")

	mr := NewManager(db, time.Second, time.Hour)
	m := mr.(*manager) //nolint:forcetypeassert

	for name, query := range map[string]func() error{
		"add":                  func() error { return m.add(addr, map[string]float64{"on": 1}, "W") },
		"remove":               func() error { return m.remove(addr) },
		"updateDeviceActivity": func() error { return m.updateDeviceActivity(addr, true) },
	} {
		t.Run(name, func(t *testing.T) {
			free := false

			mockAdapter := mockedadapter.NewAdapter(t)
			for _, method := range []string{"ThingByTopic", "ThingByAddress"} {
				mockAdapter.On(method, addr).Run(func(mock.Arguments) {
					free = m.lock.TryLock()
					if free {
						m.lock.Unlock()
					}
				}).Return(nil).Maybe()
			}

			m.ad = mockAdapter
			m.virtualServices = map[string]adapter.Service{addr: outLvlSwitchService}

			assert.Error(t, query(), "should fail with no thing found")
			assert.True(t, free, "manager lock must be free while the adapter is queried")
		})
	}
}

func TestManager_Update(t *testing.T) { //nolint:paralleltest
	cases := []struct {
		name           string
		registerDevice bool
		existingModes  map[string]float64
		expectError    bool
		expectedMode   string
		expectedLevel  float64
	}{
		{
			name:           "should fail to update when no device found",
			registerDevice: false,
			expectError:    true,
		},
		{
			name:           "should not update when device isn't initialised",
			registerDevice: true,
			existingModes:  nil,
			expectError:    false,
			expectedMode:   ModeOn,
			expectedLevel:  0,
		},
		{
			name:           "should update modes and level",
			registerDevice: true,
			existingModes:  map[string]float64{"on": 123},
			expectError:    false,
			expectedMode:   ModeOff,
			expectedLevel:  14,
		},
	}

	for _, cc := range cases { //nolint:paralleltest
		c := cc
		t.Run(c.name, func(t *testing.T) {
			defer adapterhelper.TearDownAdapter(workdir)[0](t)

			db, _ := database.NewDatabase(workdir)
			mr := NewManager(db, time.Second, time.Hour)
			m := mr.(*manager) //nolint:forcetypeassert

			if c.registerDevice {
				err := m.storage.SetDevice(addr, &Device{Modes: c.existingModes, CurrentMode: ModeOn})
				assert.NoError(t, err, "should set device")
			}

			err := m.update(addr, "off", 14)
			if c.expectError {
				assert.Error(t, err, "should fail to add a meter")
			} else {
				assert.NoError(t, err, "should add a meter")
				device, err := m.storage.Device(addr)
				assert.NoError(t, err, "should get modes")
				assert.Equal(t, c.expectedMode, device.CurrentMode, "should remove modes")
				assert.Equal(t, c.expectedLevel, device.Level, "should remove modes")
			}
		})
	}
}

func TestManager_Report(t *testing.T) { //nolint:paralleltest
	recalculationPeriod := time.Second

	cases := []struct {
		name           string
		device         *Device
		unit           numericmeter.Unit
		expectError    bool
		expectedReport float64
	}{
		{
			name:        "should return error when no device found",
			device:      nil,
			expectError: true,
		},
		{
			name: "should calculated energy and return",
			device: &Device{
				CurrentMode:       ModeOn,
				Modes:             map[string]float64{ModeOn: 100},
				Level:             1.0,
				LastTimeUpdated:   time.Now().Add(-3 * recalculationPeriod),
				Active:            true,
				AccumulatedEnergy: 200,
			},
			unit:           numericmeter.UnitKWh,
			expectError:    false,
			expectedReport: 200 + 2*recalculationPeriod.Hours()*100*1.0/1000,
		},
		{
			name: "should not recalculate energy but return cached values when device inactive",
			device: &Device{
				Active:            false,
				AccumulatedEnergy: 213,
			},
			unit:           numericmeter.UnitKWh,
			expectError:    false,
			expectedReport: 213,
		},
	}

	for _, cc := range cases { //nolint:paralleltest
		c := cc
		t.Run(c.name, func(t *testing.T) {
			defer adapterhelper.TearDownAdapter(workdir)[0](t)

			db, _ := database.NewDatabase(workdir)
			mr := NewManager(db, time.Second, time.Hour)
			m := mr.(*manager) //nolint:forcetypeassert

			if c.device != nil {
				err := m.storage.SetDevice(addr, c.device)
				assert.NoError(t, err, "should set device")
			}

			report, err := m.report(addr, c.unit)
			if c.expectError {
				assert.Error(t, err, "should fail to report")
			} else {
				assert.NoError(t, err, "should add a meter")
				assert.Equal(t, c.expectedReport, report, "unexpected report")
			}
		})
	}
}

func TestManager_ReportPerUnit(t *testing.T) {
	t.Parallel()

	device := &Device{
		Modes:             map[string]float64{ModeOn: 100},
		CurrentMode:       ModeOn,
		Level:             0.8,
		AccumulatedEnergy: 432,
	}

	cases := []struct {
		name           string
		unit           numericmeter.Unit
		expectError    bool
		expectedReport float64
	}{
		{
			name:           "should return accumulated energy when KWh provided",
			unit:           numericmeter.UnitKWh,
			expectError:    false,
			expectedReport: device.AccumulatedEnergy,
		},
		{
			name:           "should return current mode data when W provided",
			unit:           numericmeter.UnitW,
			expectError:    false,
			expectedReport: device.Modes[device.CurrentMode] * device.Level,
		},
		{
			name:        "should return error when unknown unit provided",
			unit:        numericmeter.Unit("unknown"),
			expectError: true,
		},
	}

	for _, cc := range cases {
		c := cc
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			report, err := (&manager{}).reportPerUnit(device, c.unit)
			if c.expectError {
				assert.Error(t, err, "should fail to report")
			} else {
				assert.NoError(t, err, "shouldn't fail on report")
				assert.Equal(t, c.expectedReport, report, "unexpected report")
			}
		})
	}
}

func TestManager_Reset(t *testing.T) { //nolint:paralleltest
	cases := []struct {
		name        string
		device      *Device
		expectError bool
	}{
		{
			name:        "should return error when no device found",
			device:      nil,
			expectError: true,
		},
		{
			name: "should reset accumulated energy",
			device: &Device{
				AccumulatedEnergy: 213,
			},
			expectError: false,
		},
	}

	for _, cc := range cases { //nolint:paralleltest
		c := cc
		t.Run(c.name, func(t *testing.T) {
			defer adapterhelper.TearDownAdapter(workdir)[0](t)

			db, _ := database.NewDatabase(workdir)
			mr := NewManager(db, time.Second, time.Hour)
			m := mr.(*manager) //nolint:forcetypeassert

			if c.device != nil {
				err := m.storage.SetDevice(addr, c.device)
				assert.NoError(t, err, "should set device")
			}

			err := m.reset(addr)
			if c.expectError {
				assert.Error(t, err, "should fail to report")
			} else {
				assert.NoError(t, err, "should add a meter")

				device, err := m.storage.Device(addr)
				assert.NoError(t, err, "should get a device")
				assert.Equal(t, float64(0), device.AccumulatedEnergy, "unexpected report")
			}
		})
	}
}

func TestManager_CleanOrphanedDevices(t *testing.T) { //nolint:paralleltest
	const gracePeriod = time.Hour

	stale := time.Now().Add(-2 * gracePeriod)
	beyondGrace := time.Now().Add(-2 * gracePeriod)
	withinGrace := time.Now().Add(-gracePeriod / 2)

	cases := []struct {
		name   string
		device *Device
		live   bool
		// registered mirrors a topic this process has registered, which vetoes deletion because
		// liveTopics is snapshotted before the manager lock is taken.
		registered   bool
		expectDelete bool
		expectStamp  bool
	}{
		{
			// The regression this whole task guards: nothing refreshes LastTimeUpdated while a
			// device is inactive, so a configured meter on an offline thing looks arbitrarily stale.
			name:   "should keep a stale configured device whose service is still live",
			device: &Device{Modes: map[string]float64{"on": 123}, AccumulatedEnergy: 42, LastTimeUpdated: stale},
			live:   true,
		},
		{
			// registerVirtualServices seeds these and never refreshes them; deleting one makes
			// every later cmd.meter.add on that device fail until the adapter restarts.
			name:   "should keep a stale unconfigured placeholder whose service is still live",
			device: &Device{Modes: nil, LastTimeUpdated: stale},
			live:   true,
		},
		{
			// Without clearing, a device that returns and is orphaned again years later would be
			// deleted by the first pass that notices, with no grace period at all.
			name:   "should clear a persisted orphan stamp when the service is live again",
			device: &Device{Modes: map[string]float64{"on": 123}, LastTimeUpdated: stale, OrphanedSince: &beyondGrace},
			live:   true,
		},
		{
			// The manager is built fresh here, so a stamp that only lived in memory would be lost:
			// this is the restart case, and the stored stamp is what still ages across it.
			name:         "should delete a device orphaned since before the last restart",
			device:       &Device{Modes: map[string]float64{"on": 123}, LastTimeUpdated: stale, OrphanedSince: &beyondGrace},
			expectDelete: true,
		},
		{
			name:        "should keep an orphaned device within the grace period",
			device:      &Device{Modes: map[string]float64{"on": 123}, LastTimeUpdated: stale, OrphanedSince: &withinGrace},
			expectStamp: true,
		},
		{
			// A stale device seen orphaned for the first time only starts the grace period; the
			// pass that first notices it must never be the one that deletes it.
			name:        "should stamp but keep a device seen orphaned for the first time",
			device:      &Device{Modes: map[string]float64{"on": 123}, LastTimeUpdated: stale},
			expectStamp: true,
		},
		{
			// A thing being registered right now is absent from the snapshot taken before the lock;
			// deleting its row here would break every later cmd.meter.add on it.
			name:        "should keep an orphaned device this process still has registered",
			device:      &Device{Modes: map[string]float64{"on": 123}, LastTimeUpdated: stale, OrphanedSince: &beyondGrace},
			registered:  true,
			expectStamp: true,
		},
	}

	for _, cc := range cases { //nolint:paralleltest
		c := cc
		t.Run(c.name, func(t *testing.T) {
			defer adapterhelper.TearDownAdapter(workdir)[0](t)

			db, _ := database.NewDatabase(workdir)
			mr := NewManager(db, time.Second, gracePeriod)
			m := mr.(*manager) //nolint:forcetypeassert

			assert.NoError(t, m.storage.SetDevice(addr, c.device), "should set device")

			liveTopics := make(map[string]struct{})
			if c.live {
				liveTopics[addr] = struct{}{}
			}

			if c.registered {
				m.virtualServices[addr] = meterElecService
			}

			assert.NoError(t, m.cleanOrphanedDevices(liveTopics), "should clean without error")

			device, err := m.storage.Device(addr)
			assert.NoError(t, err, "should get a device")

			if c.expectDelete {
				assert.Nil(t, device, "device should have been deleted")

				return
			}

			assert.NotNil(t, device, "device should have been kept")
			assert.Equal(t, c.device.Modes, device.Modes, "modes should survive")
			assert.Equal(t, c.device.AccumulatedEnergy, device.AccumulatedEnergy, "accumulated energy should survive")

			if c.expectStamp {
				assert.NotNil(t, device.OrphanedSince, "orphan stamp should be kept")
			} else {
				assert.Nil(t, device.OrphanedSince, "orphan stamp should be cleared")
			}
		})
	}
}

func TestManager_RecalculateEnergy(t *testing.T) {
	t.Parallel()

	recalculationPeriod := time.Second

	cases := []struct {
		name              string
		force             bool
		device            *Device
		expectError       bool
		expectedEnergy    float64
		shouldRecalculate bool
	}{
		{
			name: "should not calculate energy when device inactive",
			device: &Device{
				AccumulatedEnergy: 213,
			},
			expectError:    false,
			expectedEnergy: 213,
		},
		{
			name:  "should not recalculate energy when not forced and little time left",
			force: false,
			device: &Device{
				Active:            true,
				LastTimeUpdated:   time.Now().Add(recalculationPeriod + 100),
				AccumulatedEnergy: 331,
			},
			expectError:    false,
			expectedEnergy: 331,
		},
		{
			name:  "should recalculate energy when forced and little time left",
			force: true,
			device: &Device{
				Active:            true,
				LastTimeUpdated:   time.Now().Add(-recalculationPeriod),
				AccumulatedEnergy: 123,
				Modes: map[string]float64{
					ModeOn: 140,
				},
				CurrentMode: ModeOn,
				Level:       0.7,
			},
			expectError:       false,
			expectedEnergy:    123 + recalculationPeriod.Hours()*140*0.7/1000,
			shouldRecalculate: true,
		},
		{
			name:  "should recalculate energy when not forced and a lot of time left",
			force: false,
			device: &Device{
				Active:            true,
				LastTimeUpdated:   time.Now().Add(-4 * recalculationPeriod),
				AccumulatedEnergy: 645,
				Modes: map[string]float64{
					ModeOn: 190,
				},
				CurrentMode: ModeOn,
				Level:       0.5,
			},
			expectError:       false,
			expectedEnergy:    645 + 2*recalculationPeriod.Hours()*190*0.5/1000,
			shouldRecalculate: true,
		},
	}

	for _, cc := range cases {
		c := cc
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			recalculated, err := (&manager{}).recalculateEnergy(c.force, c.device)
			if c.expectError {
				assert.Error(t, err, "should fail to report")
			} else {
				assert.NoError(t, err, "should add a meter")
				assert.Equal(t, c.shouldRecalculate, recalculated, "unexpected recalculation status")

				e := 0.0
				if c.shouldRecalculate {
					e = recalculationPeriod.Hours() * c.device.Modes[c.device.CurrentMode] * c.device.Level / 1000
				}

				assert.InEpsilon(t, c.expectedEnergy, c.device.AccumulatedEnergy, e, "unexpected report")
			}
		})
	}
}

func TestManager_UpdateDeviceActivity(t *testing.T) { //nolint:paralleltest
	cases := []struct {
		name        string
		thing       adapter.Thing
		service     adapter.Service
		device      *Device
		expectError bool
	}{
		{
			name:        "should fail when no thing found",
			thing:       nil,
			expectError: true,
		},
		{
			name:        "should return error when device no found",
			thing:       mockedadapter.NewThing(t).WithServices("", true, []adapter.Service{outLvlSwitchService}),
			service:     outLvlSwitchService,
			device:      nil,
			expectError: true,
		},
		{
			name:    "should update the device with true",
			thing:   mockedadapter.NewThing(t).WithServices("", true, []adapter.Service{outLvlSwitchService}),
			service: outLvlSwitchService,
			device: &Device{
				Active: false,
			},
			expectError: false,
		},
	}

	for _, cc := range cases { //nolint:paralleltest
		c := cc
		t.Run(c.name, func(t *testing.T) {
			db, err := database.NewDatabase(workdir)
			assert.NoError(t, err, "should create a database")

			mr := NewManager(db, time.Second, time.Hour)
			m := mr.(*manager) //nolint:forcetypeassert

			mockedAdapter := mockedadapter.NewAdapter(t).WithThingByAddress(addr, true, c.thing)
			m.ad = mockedAdapter

			if c.service != nil {
				m.virtualServices = map[string]adapter.Service{addr: c.service}
			}

			if c.device != nil {
				err := m.storage.SetDevice(addr, c.device)
				assert.NoError(t, err, "should set device")
			}

			err = m.updateDeviceActivity(addr, true)
			if c.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				device, err := m.storage.Device(addr)

				assert.NoError(t, err)
				assert.Equal(t, true, device.Active)
			}
		})
	}
}

func TestManager_RegisterDevice(t *testing.T) { //nolint:paralleltest
	cases := []struct {
		name        string
		thing       adapter.Thing
		adapter     adapter.Adapter
		deviceKey   string
		device      *Device
		expectError bool
	}{
		{
			name: "should not error when didn't find any services to create",
			thing: mockedadapter.NewThing(t).
				WithInclusionReported(&fimptype.ThingInclusionReport{Address: addr, Groups: []string{"ch1"}}, true).
				WithServices("", true, []adapter.Service{}),
			adapter:     mockedadapter.NewAdapter(t),
			expectError: false,
		},
		{
			name: "should nor return error if skipped services without groups",
			thing: mockedadapter.NewThing(t).
				WithInclusionReported(&fimptype.ThingInclusionReport{Address: addr, Groups: []string{"ch1"}}, true).
				WithServices("", true, []adapter.Service{
					outlvlswitch.NewService(nil, &outlvlswitch.Config{
						Specification: &fimptype.Service{
							Name:    outlvlswitch.OutLvlSwitch,
							Address: "/rt:dev/rn:test/ad:test/sv:out_level_switch/ad:test",
							Groups:  []string{},
						},
					}),
				}),
		},
		{
			name: "should avoid updating if virtual meter already exists",
			thing: mockedadapter.NewThing(t).
				WithInclusionReported(&fimptype.ThingInclusionReport{Address: addr, Groups: []string{"ch1"}}, true).
				WithServices(VirtualMeterElec, true, []adapter.Service{
					&service{
						Service: adapter.NewService(nil, &fimptype.Service{
							Name:    VirtualMeterElec,
							Address: "/rt:dev/rn:test/ad:test/sv:virtual_meter_elec/ad:test_ch1",
							Groups:  []string{"ch1"},
						}),
					},
				}).
				WithServices("", true, []adapter.Service{outLvlSwitchServiceFullAddr}),
			adapter: mockedadapter.NewAdapter(t).
				WithName("test", true).
				WithName("test", true).
				WithAddress(addr, true).
				WithAddress(addr, true),
			expectError: false,
		},
		{
			name: "should return error when update fails",
			thing: mockedadapter.NewThing(t).
				WithInclusionReported(&fimptype.ThingInclusionReport{Address: addr, Groups: []string{"ch1"}}, true).
				WithServices(VirtualMeterElec, true, []adapter.Service{}).
				WithServices("", true, []adapter.Service{outLvlSwitchServiceFullAddr}).
				WithUpdate(true, errors.New("some")),
			adapter: mockedadapter.NewAdapter(t).
				WithName("test", true).
				WithName("test", true).
				WithAddress(addr, true).
				WithAddress(addr, true),
			expectError: true,
		},
		{
			name: "should update thing if device already exists",
			thing: mockedadapter.NewThing(t).
				WithInclusionReported(&fimptype.ThingInclusionReport{Address: addr, Groups: []string{"ch1"}}, true).
				WithServices(VirtualMeterElec, true, []adapter.Service{}).
				WithServices("", true, []adapter.Service{outLvlSwitchServiceFullAddr}).
				WithUpdate(true, nil).
				WithUpdate(true, nil),
			adapter: mockedadapter.NewAdapter(t).
				WithName("test", true).
				WithName("test", true).
				WithAddress(addr, true).
				WithAddress(addr, true),
			deviceKey: "/rt:dev/rn:test/ad:test/sv:virtual_meter_elec/ad:test_ch1",
			device: &Device{
				Modes: map[string]float64{ModeOn: 123},
			},
			expectError: false,
		},
		{
			name: "should update uninitialized device",
			thing: mockedadapter.NewThing(t).
				WithInclusionReported(&fimptype.ThingInclusionReport{Address: addr, Groups: []string{"ch1"}}, true).
				WithServices(VirtualMeterElec, true, []adapter.Service{}).
				WithServices("", true, []adapter.Service{outLvlSwitchServiceFullAddr}).
				WithUpdate(true, nil),
			adapter: mockedadapter.NewAdapter(t).
				WithName("test", true).
				WithName("test", true).
				WithAddress(addr, true).
				WithAddress(addr, true),
			deviceKey: "/rt:dev/rn:test/ad:test/sv:virtual_meter_elec/ad:test_ch1",
			device: &Device{
				Modes: nil,
			},
			expectError: false,
		},
	}

	for _, cc := range cases { //nolint:paralleltest
		c := cc
		t.Run(c.name, func(t *testing.T) {
			defer adapterhelper.TearDownAdapter(workdir)[0](t)

			db, err := database.NewDatabase(workdir)
			assert.NoError(t, err, "should create a database")

			mr := NewManager(db, time.Second, time.Hour)
			m := mr.(*manager) //nolint:forcetypeassert
			m.ad = c.adapter

			m.virtualServices = make(map[string]adapter.Service)

			// pre-creating the state of the device represented by services.
			if c.device != nil {
				err := m.storage.SetDevice(c.deviceKey, c.device)
				assert.NoError(t, err, "should set device")
			}

			err = mr.RegisterThing(c.thing, nil)
			if c.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// validating the device data remains intact byt key equal to virtual meter address.
				if c.device != nil && c.deviceKey != "" {
					device, err := m.storage.Device(c.deviceKey)
					assert.NoError(t, err)

					assert.Equal(t, c.device.Modes, device.Modes)
					assert.Equal(t, c.device.Active, device.Active)
				}
			}
		})
	}
}

func TestManager_vmsAddressFromTopic(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                string
		topic               string
		adapter             adapter.Adapter
		expectedServiceAddr string
	}{
		{
			name:                "should error when failed to parse topic",
			topic:               "invalid/invalid",
			expectedServiceAddr: "",
		},
		{
			name:                "should error when failed to find thing by topic",
			topic:               "rt:dev/rn:test/ad:1/sv:meter_elec/ad:1",
			adapter:             mockedadapter.NewAdapter(t).WithThingByTopic("rt:dev/rn:test/ad:1/sv:meter_elec/ad:1", true, nil),
			expectedServiceAddr: "",
		},
		{
			name:  "should error when failed to find any vms services",
			topic: "rt:dev/rn:test/ad:1/sv:meter_elec/ad:1",
			adapter: mockedadapter.NewAdapter(t).WithThingByTopic(
				"rt:dev/rn:test/ad:1/sv:meter_elec/ad:1",
				true,
				mockedadapter.NewThing(t).WithServices(VirtualMeterElec, true, nil),
			),
			expectedServiceAddr: "",
		},
		{
			name:  "should error when failed to find vms with matching service address",
			topic: "rt:dev/rn:test/ad:1/sv:meter_elec/ad:1",
			adapter: mockedadapter.NewAdapter(t).WithThingByTopic(
				"rt:dev/rn:test/ad:1/sv:meter_elec/ad:1",
				true,
				mockedadapter.NewThing(t).WithServices(
					VirtualMeterElec,
					true,
					[]adapter.Service{mockedadapter.NewService(t).WithTopic(true, "rt:dev/rn:test/ad:1/sv:virtual_meter_elec/ad:232")},
				),
			),
			expectedServiceAddr: "",
		},
		{
			name:  "should return service full address when found matching vms",
			topic: "rt:dev/rn:test/ad:1/sv:meter_elec/ad:1_ch",
			adapter: mockedadapter.NewAdapter(t).WithThingByTopic(
				"rt:dev/rn:test/ad:1/sv:meter_elec/ad:1_ch",
				true,
				mockedadapter.NewThing(t).WithServices(
					VirtualMeterElec,
					true,
					[]adapter.Service{
						mockedadapter.NewService(t).
							WithTopic(true, "rt:dev/rn:test/ad:1/sv:virtual_meter_elec/ad:1_ch").
							WithTopic(true, "rt:dev/rn:test/ad:1/sv:virtual_meter_elec/ad:1_ch"),
					},
				),
			),
			expectedServiceAddr: "rt:dev/rn:test/ad:1/sv:virtual_meter_elec/ad:1_ch",
		},
	}

	for _, cc := range cases {
		c := cc
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			m := manager{ad: c.adapter}
			foundAddr, err := m.vmsAddressFromTopic(c.topic)

			if c.expectedServiceAddr != "" {
				assert.NoError(t, err)
				assert.Equal(t, c.expectedServiceAddr, foundAddr)
			} else {
				assert.Error(t, err)
				assert.Equal(t, "", foundAddr)
			}
		})
	}
}
