package virtualmeter //nolint:testpackage

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/database"
	"github.com/futurehomeno/cliffhanger/event"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
)

const (
	workdir = "../../../testdata/adapter/test_adapter"
)

func TestHandlerLevelEvent(t *testing.T) { //nolint:paralleltest
	lvlService := outlvlswitch.NewService(nil,
		&outlvlswitch.Config{
			Specification: &fimptype.Service{
				Name:    outlvlswitch.OutLvlSwitch,
				Address: "",
				Props: map[string]interface{}{
					outlvlswitch.PropertyMaxLvl: 100,
				},
			},
		},
	)

	vms := NewService(nil,
		&Config{
			Specification: &fimptype.Service{
				Name:    VirtualMeterElec,
				Address: "",
			},
			ManagerWrapper: NewManagerWrapper(nil, 0, time.Hour),
		},
	)

	testCases := []struct {
		name           string
		thing          adapter.Thing
		levelEvent     *outlvlswitch.LevelEvent
		expectedDevice *Device
	}{
		{
			name: "level shouldn't call update if level hasn't changed and update isn't required",
			thing: mockedadapter.NewThing(t).
				WithServices(VirtualMeterElec, true, []adapter.Service{vms}).
				WithServices(outlvlswitch.OutLvlSwitch, true, []adapter.Service{lvlService}),
			levelEvent: &outlvlswitch.LevelEvent{
				Level:        0,
				ServiceEvent: adapter.NewServiceEvent("type", false),
			},
			expectedDevice: nil,
		},
		{
			name: "level should call update if level has changed, mode on",
			thing: mockedadapter.NewThing(t).
				WithServices(outlvlswitch.OutLvlSwitch, true, []adapter.Service{lvlService}).
				WithServices(VirtualMeterElec, true, []adapter.Service{vms}),
			levelEvent: &outlvlswitch.LevelEvent{
				Level:        13,
				ServiceEvent: adapter.NewServiceEvent("type", true),
			},
			expectedDevice: &Device{
				Modes:       map[string]float64{ModeOn: 100},
				CurrentMode: ModeOn,
				Level:       0.13,
			},
		},
		{
			name: "level should call update if level has changed, mode off",
			thing: mockedadapter.NewThing(t).
				WithServices(outlvlswitch.OutLvlSwitch, true, []adapter.Service{lvlService}).
				WithServices(VirtualMeterElec, true, []adapter.Service{vms}),
			levelEvent: &outlvlswitch.LevelEvent{
				Level:        0,
				ServiceEvent: adapter.NewServiceEvent("type", true),
			},
			expectedDevice: &Device{
				Modes:       map[string]float64{ModeOn: 100},
				Level:       0,
				CurrentMode: ModeOff,
			},
		},
	}

	for _, tc := range testCases { //nolint:paralleltest
		v := tc
		t.Run(tc.name, func(t *testing.T) {
			defer adapterhelper.TearDownAdapter(workdir)[0](t)

			db, _ := database.NewDatabase(workdir)
			mr := NewManagerWrapper(db, 0, time.Hour)
			m := mr.(*manager) //nolint:forcetypeassert

			if v.thing != nil {
				m.ad = mockedadapter.NewAdapter(t).WithThingByTopic("", false, v.thing)
				m.virtualServices[addr] = outLvlSwitchService
			}

			if v.expectedDevice != nil {
				err := m.storage.SetDevice("", &Device{Modes: map[string]float64{ModeOn: 100}})
				assert.NoError(t, err, "should set a device at start")
			}

			handlers := NewHandlers(mr)
			eventManager := event.NewManager()
			listener := event.NewListener(eventManager, handlers...)

			err := listener.Start()
			assert.NoError(t, err, "listener should start")

			time.Sleep(100 * time.Millisecond)
			defer listener.Stop() //nolint:errcheck

			if v.levelEvent != nil {
				eventManager.Publish(v.levelEvent)
			}

			time.Sleep(50 * time.Millisecond)
			d, err := m.storage.Device("")
			assert.NoError(t, err, "should return a device")
			assert.Equal(t, v.expectedDevice, d, "should return the same device as was saved")
		})
	}
}

func TestHandlerConnectivityEvent(t *testing.T) { //nolint:paralleltest
	testCases := []struct {
		name              string
		thing             adapter.Thing
		connectivityEvent *adapter.ConnectivityEvent
		expectedDevice    *Device
	}{
		{
			name:  "connectivity event should update connectivity",
			thing: mockedadapter.NewThing(t).WithServices("", true, []adapter.Service{outLvlSwitchService}),
			connectivityEvent: &adapter.ConnectivityEvent{
				ThingEvent:   adapter.NewThingEvent("ad1", nil),
				Connectivity: &adapter.ConnectivityDetails{ConnectionStatus: adapter.ConnectionStatusUp},
			},
			expectedDevice: &Device{
				Modes:  make(map[string]float64),
				Active: true,
			},
		},
	}

	for _, tc := range testCases { //nolint:paralleltest
		v := tc
		t.Run(tc.name, func(t *testing.T) {
			defer adapterhelper.TearDownAdapter(workdir)[0](t)

			db, _ := database.NewDatabase(workdir)
			mr := NewManagerWrapper(db, 0, time.Hour)
			m := mr.(*manager) //nolint:forcetypeassert

			if v.thing != nil {
				m.ad = mockedadapter.NewAdapter(t).WithThingByAddress("ad1", true, v.thing)
				m.virtualServices[addr] = outLvlSwitchService
			}

			if v.expectedDevice != nil {
				err := m.storage.SetDevice(addr, &Device{Modes: make(map[string]float64)})
				assert.NoError(t, err, "should set a device at start")
			}

			handlers := NewHandlers(mr)
			eventManager := event.NewManager()
			listener := event.NewListener(eventManager, handlers...)

			err := listener.Start()
			assert.NoError(t, err, "listener should start")

			time.Sleep(100 * time.Millisecond)
			defer listener.Stop() //nolint:errcheck

			if v.connectivityEvent != nil {
				eventManager.Publish(v.connectivityEvent)
			}

			time.Sleep(50 * time.Millisecond)
			d, err := m.storage.Device(addr)
			assert.NoError(t, err, "should return a device")
			assert.Equal(t, v.expectedDevice, d, "should return the same device as was saved")
		})
	}
}
