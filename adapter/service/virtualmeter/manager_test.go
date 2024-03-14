package virtualmeter //nolint:testpackage

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/database"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
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
			name:              "should succeed and updated modes",
			configuredService: outLvlSwitchService,
			existingDevice:    &Device{Modes: map[string]float64{"on": 123}},
			mockedThing:       mockedadapter.NewThing(t),
			teardown:          adapterhelper.TearDownAdapter(workdir)[0],
			expectError:       false,
		},
		{
			name:              "should succeed and updated modes, and send inclusion report",
			configuredService: outLvlSwitchService,
			existingDevice:    &Device{Modes: nil},
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

			mockAdapter := mockedadapter.NewAdapter(t)
			if c.configuredService != nil {
				mockAdapter = mockAdapter.WithThingByTopic(addr, true, c.mockedThing)
			}

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

			mockAdapter := mockedadapter.NewAdapter(t)
			if c.configuredService != nil {
				mockAdapter = mockAdapter.WithThingByTopic(addr, true, c.mockedThing)
			}

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
