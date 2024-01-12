package virtualmeter_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/adapter/service/virtualmeter"
	"github.com/futurehomeno/cliffhanger/event"
	mockedvirtualmeter "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/virtualmeter"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		levelEvent        *outlvlswitch.LevelEvent
		connectivityEvent *adapter.ConnectivityEvent
		vmeterMock        *mockedvirtualmeter.ManagerFull
	}{
		{
			name: "level shouldn't call update if level hasn't changed and update isn't required",
			levelEvent: &outlvlswitch.LevelEvent{
				Level:        0,
				ServiceEvent: adapter.NewServiceEvent("type", false),
			},
			vmeterMock: mockedvirtualmeter.NewFullManager(t).WithUpdateRequired(false, true, ""),
		},
		{
			name: "level should call update if level has changed, mode on",
			levelEvent: &outlvlswitch.LevelEvent{
				Level:        0.13,
				ServiceEvent: adapter.NewServiceEvent("type", true),
			},
			vmeterMock: mockedvirtualmeter.NewFullManager(t).WithUpdate(nil, true, "", virtualmeter.ModeOn, 0.13),
		},
		{
			name: "level should call update if level has changed, mode off",
			levelEvent: &outlvlswitch.LevelEvent{
				Level:        0,
				ServiceEvent: adapter.NewServiceEvent("type", true),
			},
			vmeterMock: mockedvirtualmeter.NewFullManager(t).WithUpdate(nil, true, "", virtualmeter.ModeOff, 0.0),
		},
		{
			name: "connectivity event should update connectivity",
			connectivityEvent: &adapter.ConnectivityEvent{
				ThingEvent:   adapter.NewThingEvent("ad1", nil),
				Connectivity: &adapter.ConnectivityDetails{ConnectionStatus: adapter.ConnectionStatusUp},
			},
			vmeterMock: mockedvirtualmeter.NewFullManager(t).WithUpdateDeviceActivity(nil, true, "ad1", true),
		},
	}

	for _, tc := range testCases {
		v := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler := virtualmeter.NewHandler(v.vmeterMock)
			eventManager := event.NewManager()
			listener := event.NewListener(eventManager, handler)

			err := listener.Start()
			assert.NoError(t, err, "listener should start")

			time.Sleep(100 * time.Millisecond)
			defer listener.Stop() //nolint:errcheck

			if v.levelEvent != nil {
				eventManager.Publish(v.levelEvent)
			}

			if v.connectivityEvent != nil {
				eventManager.Publish(v.connectivityEvent)
			}
		})
	}
}
