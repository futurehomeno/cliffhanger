package virtualmeter_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/virtualmeter"
	"github.com/futurehomeno/cliffhanger/database"
	"github.com/futurehomeno/cliffhanger/event"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
)

const (
	addr = "test:addr"
)

func TestVirtualMeterManager_Add(t *testing.T) { //nolint:paralleltest
	cases := []struct {
		name        string
		modes       map[string]float64
		mockedThing adapter.Thing
		teardown    func(t *testing.T)
		expectError bool
	}{
		{
			name:        "should failed to add modes when thing is not found",
			modes:       map[string]float64{"on": 100},
			mockedThing: nil,
			teardown:    adapterhelper.TearDownAdapter(workdir)[0],
			expectError: true,
		},
		{
			name:        "should add modes when thing is found",
			modes:       map[string]float64{"on": 113},
			mockedThing: mockedadapter.NewThing(t).WithUpdate(true, true, nil),
			teardown:    func(t *testing.T) { t.Helper() },
			expectError: false,
		},
		{
			name:        "should add modes but updates thing when registering device",
			modes:       map[string]float64{"on": 116},
			mockedThing: mockedadapter.NewThing(t).WithUpdate(false, true, nil),
			teardown:    adapterhelper.TearDownAdapter(workdir)[0],
			expectError: false,
		},
	}

	for _, cc := range cases { //nolint:paralleltest
		c := cc
		t.Run(c.name, func(t *testing.T) {
			defer c.teardown(t)

			manager, _ := Setup(t, addr, c.mockedThing, time.Second)

			err := manager.Add(addr, c.modes, "W")

			if c.expectError {
				assert.Error(t, err, "should fail to add a meter")
			} else {
				assert.NoError(t, err, "should add a meter")

				modes, err := manager.Modes(addr)
				assert.NoError(t, err, "should get modes")
				assert.Equal(t, c.modes, modes, "should set modes when no eror")
			}
		})
	}
}

func TestVirtualMeterManager_Remove(t *testing.T) { //nolint:paralleltest
	cases := []struct {
		name        string
		modes       map[string]float64
		mockedThing adapter.Thing
		expectError bool
	}{
		{
			name:        "should failed to remove modes when thing is not found",
			mockedThing: nil,
			expectError: true,
		},
		{
			name:        "should remove modes when thing is found",
			mockedThing: mockedadapter.NewThing(t).WithUpdate(true, true, nil),
			modes:       nil,
			expectError: false,
		},
	}

	for _, cc := range cases { //nolint:paralleltest
		c := cc
		t.Run(c.name, func(t *testing.T) {
			defer adapterhelper.TearDownAdapter(workdir)[0](t)

			manager, _ := Setup(t, addr, c.mockedThing, time.Second)

			err := manager.Remove(addr)

			if c.expectError {
				assert.Error(t, err, "should fail to add a meter")
			} else {
				assert.NoError(t, err, "should add a meter")

				modes, err := manager.Modes(addr)
				assert.NoError(t, err, "should get modes")
				assert.Equal(t, c.modes, modes, "should remove modes")
			}
		})
	}
}

func TestVirtualMeterManager_Update(t *testing.T) { //nolint:paralleltest
	recalculationPeriod := time.Millisecond * 1000

	cases := []struct {
		name           string
		mode           string
		level          float64
		activateDevice bool
		awaitPeriod    time.Duration
		expectedReport float64
	}{
		{
			name:           "should return zero value when device isn't active",
			mode:           "on",
			level:          0.3,
			awaitPeriod:    recalculationPeriod * 0,
			expectedReport: 0.0,
		},
		{
			name:           "should return zero value when little time elapsed",
			mode:           "on",
			level:          0.5,
			activateDevice: true,
			awaitPeriod:    recalculationPeriod * -1,
			expectedReport: 0.0,
		},
		{
			name:           "should calculate power usage over 2 * recalculation period when more them 2 * recalculation period elapsed",
			mode:           "on",
			level:          0.5,
			activateDevice: true,
			awaitPeriod:    recalculationPeriod * 3,
			expectedReport: 100 * 0.5 * 2 * recalculationPeriod.Hours() / 1000,
		},
		{
			name:           "should calculate power usage over 2 * recalculation period when more them 2 * recalculation period elapsed №2",
			mode:           "on",
			level:          0.8,
			activateDevice: true,
			awaitPeriod:    recalculationPeriod * 3,
			expectedReport: 100 * 0.8 * 2 * recalculationPeriod.Hours() / 1000,
		},
	}

	for _, cc := range cases { //nolint:paralleltest
		c := cc
		t.Run(c.name, func(t *testing.T) {
			defer adapterhelper.TearDownAdapter(workdir)[0](t)

			mockedThing := mockedadapter.NewThing(t).WithUpdate(true, true, nil)
			manager, mockedAdapter := Setup(t, addr, mockedThing, recalculationPeriod)

			err := manager.Add(addr, map[string]float64{"on": 100, "off": 0}, "W")
			assert.NoError(t, err, "should add a device")

			if c.activateDevice {
				eventManager := event.NewManager()
				listener := event.NewListener(eventManager, virtualmeter.NewHandler(manager))
				_ = listener.Start()

				mockedAdapter.On("ThingByAddress", addr).Return(mockedThing).Once()

				service := adapter.NewService(nil, &fimptype.Service{Address: addr})
				mockedThing.On("Services", "").Return([]adapter.Service{service})

				time.Sleep(time.Millisecond * 50)
				eventManager.Publish(&adapter.ConnectivityEvent{
					ThingEvent:   adapter.NewThingEvent(addr, nil),
					Connectivity: &adapter.ConnectivityDetails{ConnectionStatus: adapter.ConnectionStatusUp},
				})
				time.Sleep(time.Millisecond * 50)
			}

			err = manager.Update(addr, c.mode, c.level)
			assert.NoError(t, err, "should update a device")
			assert.Equal(t, false, manager.UpdateRequired(addr), "shouldn't require an update after update")

			time.Sleep(c.awaitPeriod)

			value, err := manager.Report(addr, numericmeter.UnitKWh)
			assert.NoError(t, err, "should report a value")
			assert.Equal(t, c.expectedReport, value, "should report a value")
		})
	}
}

func Setup(t *testing.T, addr string, thing adapter.Thing, period time.Duration) (virtualmeter.Manager, *mockedadapter.Adapter) {
	t.Helper()

	db, _ := database.NewDatabase(workdir)

	manager := virtualmeter.NewVirtualMeterManager(db, period)
	mockedAdapter := mockedadapter.NewAdapter(t).WithThingByTopic("test:addr", true, thing)
	manager.WithAdapter(mockedAdapter)

	err := manager.RegisterDevice(thing, addr, nil, &fimptype.Service{})
	assert.NoError(t, err, "should register device")

	return manager, mockedAdapter
}
