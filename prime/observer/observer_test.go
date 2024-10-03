package observer_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/event"
	"github.com/futurehomeno/cliffhanger/prime"
	"github.com/futurehomeno/cliffhanger/prime/observer"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestObserver(t *testing.T) { //nolint:paralleltest
	var (
		testObserver     observer.Observer
		testEventManager event.Manager
	)

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:  "Observer",
				Setup: setupObserverTest(&testObserver, &testEventManager, 5*time.Second),
				Nodes: []*suite.Node{
					{
						Name: "Initialize observer on startup",
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:cmd/rt:app/rn:vinculum/ad:1", prime.CmdPD7Request, prime.ServiceName, &prime.Request{Cmd: prime.CmdGet, Param: &prime.RequestParam{Components: []string{prime.ComponentDevice, prime.ComponentThing, prime.ComponentRoom, prime.ComponentArea}}}).ReplyWith(
								fimpgo.NewObjectMessage(prime.EvtPD7Response, prime.ServiceName, &prime.Response{ParamRaw: map[string]json.RawMessage{
									prime.ComponentDevice: json.RawMessage(`[{"id":1}]`),
									prime.ComponentThing:  json.RawMessage(`[{"id":1}]`),
									prime.ComponentRoom:   json.RawMessage(`[{"id":1}]`),
									prime.ComponentArea:   json.RawMessage(`[{"id":1}]`),
								}}, nil, nil, nil)),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								assert.NotNil(t, <-testEventManager.WaitFor(5*time.Second, observer.WaitForRefresh(prime.ComponentDevice)))

								devices, err := testObserver.GetDevices()

								assert.NoError(t, err)
								assert.Len(t, devices, 1)

								things, err := testObserver.GetThings()

								assert.NoError(t, err)
								assert.Len(t, things, 1)

								rooms, err := testObserver.GetRooms()

								assert.NoError(t, err)
								assert.Len(t, rooms, 1)

								areas, err := testObserver.GetAreas()

								assert.NoError(t, err)
								assert.Len(t, areas, 1)
							},
						},
					},
					{
						Name: "add new device",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentDevice,
							ParamRaw:  json.RawMessage(`{"id":2, "client": {"name": "test device name"}}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForDeviceChange()))

								devices, err := testObserver.GetDevices()

								assert.NoError(t, err)
								assert.Len(t, devices, 2)
							},
						},
					},
					{
						Name: "add existing device",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentDevice,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForDeviceChange()))

								devices, err := testObserver.GetDevices()

								assert.NoError(t, err)
								assert.Len(t, devices, 2)
							},
						},
					},
					{
						Name: "Edit existing device",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentDevice,
							ParamRaw:  json.RawMessage(`{"id":2, "client": {"name": "test device new name"}}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								go func() {
									assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForDeviceChange()))
								}()

								eventDeviceEdited := <-testEventManager.WaitFor(time.Second, observer.WaitForComponentEdit(prime.ComponentDevice))
								editEvent, ok := eventDeviceEdited.(*observer.EditComponentEvent)
								require.True(t, ok)
								assert.Equal(t, 2, editEvent.ID)
								assert.Equal(t, "test device new name", editEvent.Change.New)
								assert.Equal(t, "test device name", editEvent.Change.Old)
								assert.Equal(t, prime.ComponentDevice, editEvent.ComponentName)

								devices, err := testObserver.GetDevices()

								assert.NoError(t, err)
								assert.Len(t, devices, 2)
							},
						},
					},
					{
						Name: "Edit new device",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentDevice,
							ParamRaw:  json.RawMessage(`{"id":3, "client": {"name": "device nr 3"}}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForDeviceChange()))

								devices, err := testObserver.GetDevices()

								assert.NoError(t, err)
								assert.Len(t, devices, 3)
							},
						},
					},
					{
						Name: "Delete device",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdDelete,
							Component: prime.ComponentDevice,
							ID:        3,
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								go func() {
									assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForDeviceChange()))
								}()

								eventDeletedDevice := <-testEventManager.WaitFor(time.Second, observer.WaitForComponentDeletion(prime.ComponentDevice))
								deleteEvent, ok := eventDeletedDevice.(*observer.DeleteComponentEvent)
								require.True(t, ok)

								device, ok := deleteEvent.Component.(*prime.Device)
								require.True(t, ok)
								assert.Equal(t, 3, device.ID)
								assert.Equal(t, "device nr 3", *(*device).Client.Name)

								devices, err := testObserver.GetDevices()

								assert.NoError(t, err)
								assert.Len(t, devices, 2)
							},
						},
					},
					{
						Name: "add new thing",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentThing,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForThingChange()))

								things, err := testObserver.GetThings()

								assert.NoError(t, err)
								assert.Len(t, things, 2)
							},
						},
					},
					{
						Name: "add existing thing",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentThing,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForThingChange()))

								things, err := testObserver.GetThings()

								assert.NoError(t, err)
								assert.Len(t, things, 2)
							},
						},
					},
					{
						Name: "Edit existing thing",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentThing,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForThingChange()))

								things, err := testObserver.GetThings()

								assert.NoError(t, err)
								assert.Len(t, things, 2)
							},
						},
					},
					{
						Name: "Edit new thing",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentThing,
							ParamRaw:  json.RawMessage(`{"id":3}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForThingChange()))

								things, err := testObserver.GetThings()

								assert.NoError(t, err)
								assert.Len(t, things, 3)
							},
						},
					},
					{
						Name: "Delete thing",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdDelete,
							Component: prime.ComponentThing,
							ID:        3,
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForThingChange()))

								things, err := testObserver.GetThings()

								assert.NoError(t, err)
								assert.Len(t, things, 2)
							},
						},
					},
					{
						Name: "add new room",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentRoom,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForRoomChange()))

								rooms, err := testObserver.GetRooms()

								assert.NoError(t, err)
								assert.Len(t, rooms, 2)
							},
						},
					},
					{
						Name: "add existing room",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentRoom,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForRoomChange()))

								rooms, err := testObserver.GetRooms()

								assert.NoError(t, err)
								assert.Len(t, rooms, 2)
							},
						},
					},
					{
						Name: "Edit existing room",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentRoom,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForRoomChange()))

								rooms, err := testObserver.GetRooms()

								assert.NoError(t, err)
								assert.Len(t, rooms, 2)
							},
						},
					},
					{
						Name: "Edit new room",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentRoom,
							ParamRaw: json.RawMessage(`{
								"id":3,
								"alias": "basement test",
								"client": {
									"name": "basement client name test"
								},
								"outside": false,
								"type": "basement"
							}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForRoomChange()))

								rooms, err := testObserver.GetRooms()

								assert.NoError(t, err)
								assert.Len(t, rooms, 3)
							},
						},
					},
					{
						Name: "Delete room",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdDelete,
							Component: prime.ComponentRoom,
							ID:        3,
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								go func() {
									assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForRoomChange()))
								}()

								eventDeletedRoom := <-testEventManager.WaitFor(time.Second, observer.WaitForComponentDeletion(prime.ComponentRoom))
								deleteEvent, ok := eventDeletedRoom.(*observer.DeleteComponentEvent)
								require.True(t, ok)

								room, ok := deleteEvent.Component.(*prime.Room)
								require.True(t, ok)
								assert.Equal(t, 3, room.ID)
								assert.Equal(t, "basement test", room.Alias)
								assert.Equal(t, "basement", *(*room).Type)
								assert.Equal(t, "basement client name test", *(*room).Client.Name)

								rooms, err := testObserver.GetRooms()

								assert.NoError(t, err)
								assert.Len(t, rooms, 2)
							},
						},
					},
					{
						Name: "Added new area",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentArea,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForAreaChange()))

								areas, err := testObserver.GetAreas()

								assert.NoError(t, err)
								assert.Len(t, areas, 2)
							},
						},
					},
					{
						Name: "add existing area",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentArea,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForAreaChange()))

								areas, err := testObserver.GetAreas()

								assert.NoError(t, err)
								assert.Len(t, areas, 2)
							},
						},
					},
					{
						Name: "Edit existing area",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentArea,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForAreaChange()))

								areas, err := testObserver.GetAreas()

								assert.NoError(t, err)
								assert.Len(t, areas, 2)
							},
						},
					},
					{
						Name: "Edit new area",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentArea,
							ParamRaw:  json.RawMessage(`{"id":3}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForAreaChange()))

								areas, err := testObserver.GetAreas()

								assert.NoError(t, err)
								assert.Len(t, areas, 3)
							},
						},
					},
					{
						Name: "Delete area",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdDelete,
							Component: prime.ComponentArea,
							ID:        3,
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForAreaChange()))

								areas, err := testObserver.GetAreas()

								assert.NoError(t, err)
								assert.Len(t, areas, 2)
							},
						},
					},
					{
						Name:    "Corrupted notification",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, json.RawMessage(`{"cmd":1}`)),
					},
					{
						Name: "Unobserved notification",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdSet,
							Component: prime.ComponentState,
							ID:        1,
						}),
					},
					{
						Name: "Failed add new device",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentDevice,
							ParamRaw:  json.RawMessage(`{"id":"2"}`),
						}),
					},
					{
						Name: "Failed edit device",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentDevice,
							ParamRaw:  json.RawMessage(`{"id":"2"}`),
						}),
					},
					{
						Name: "Failed delete device",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdDelete,
							Component: prime.ComponentDevice,
							ID:        "A",
						}),
					},
					{
						Name: "Failed add new thing",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentThing,
							ParamRaw:  json.RawMessage(`{"id":"2"}`),
						}),
					},
					{
						Name: "Failed edit thing",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentThing,
							ParamRaw:  json.RawMessage(`{"id":"2"}`),
						}),
					},
					{
						Name: "Failed delete thing",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdDelete,
							Component: prime.ComponentThing,
							ID:        "a",
						}),
					},
					{
						Name: "Failed add new room",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentRoom,
							ParamRaw:  json.RawMessage(`{"id":"2"}`),
						}),
					},
					{
						Name: "Failed edit room",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentRoom,
							ParamRaw:  json.RawMessage(`{"id":"2"}`),
						}),
					},
					{
						Name: "Failed delete room",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdDelete,
							Component: prime.ComponentRoom,
							ID:        "A",
						}),
					},
					{
						Name: "Failed add new area",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentArea,
							ParamRaw:  json.RawMessage(`{"id":"2"}`),
						}),
					},
					{
						Name: "Failed edit area",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentArea,
							ParamRaw:  json.RawMessage(`{"id":"2"}`),
						}),
					},
					{
						Name: "Failed delete area",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdDelete,
							Component: prime.ComponentArea,
							ID:        "A",
						}),
					},
					suite.SleepNode(50 * time.Millisecond), // sleeping to allow observer to process all incoming messages
					{
						Name: "Failed lazy load on getting devices",
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:cmd/rt:app/rn:vinculum/ad:1", prime.CmdPD7Request, prime.ServiceName, &prime.Request{Cmd: prime.CmdGet, Param: &prime.RequestParam{Components: []string{prime.ComponentDevice, prime.ComponentThing, prime.ComponentRoom, prime.ComponentArea}}}).ReplyWith(
								fimpgo.NewObjectMessage(prime.EvtPD7Response, prime.ServiceName, json.RawMessage(`{"cmd":1}`), nil, nil, nil)),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								_, err := testObserver.GetDevices()

								assert.Error(t, err)
							},
						},
					},
					{
						Name: "Failed lazy load on getting things",
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:cmd/rt:app/rn:vinculum/ad:1", prime.CmdPD7Request, prime.ServiceName, &prime.Request{Cmd: prime.CmdGet, Param: &prime.RequestParam{Components: []string{prime.ComponentDevice, prime.ComponentThing, prime.ComponentRoom, prime.ComponentArea}}}).ReplyWith(
								fimpgo.NewObjectMessage(prime.EvtPD7Response, prime.ServiceName, json.RawMessage(`{"cmd":1}`), nil, nil, nil)),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								_, err := testObserver.GetThings()

								assert.Error(t, err)
							},
						},
					},
					{
						Name: "Failed lazy load on getting rooms",
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:cmd/rt:app/rn:vinculum/ad:1", prime.CmdPD7Request, prime.ServiceName, &prime.Request{Cmd: prime.CmdGet, Param: &prime.RequestParam{Components: []string{prime.ComponentDevice, prime.ComponentThing, prime.ComponentRoom, prime.ComponentArea}}}).ReplyWith(
								fimpgo.NewObjectMessage(prime.EvtPD7Response, prime.ServiceName, json.RawMessage(`{"cmd":1}`), nil, nil, nil)),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								_, err := testObserver.GetRooms()

								assert.Error(t, err)
							},
						},
					},
					{
						Name: "Failed lazy load on getting areas",
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:cmd/rt:app/rn:vinculum/ad:1", prime.CmdPD7Request, prime.ServiceName, &prime.Request{Cmd: prime.CmdGet, Param: &prime.RequestParam{Components: []string{prime.ComponentDevice, prime.ComponentThing, prime.ComponentRoom, prime.ComponentArea}}}).ReplyWith(
								fimpgo.NewObjectMessage(prime.EvtPD7Response, prime.ServiceName, json.RawMessage(`{"cmd":1}`), nil, nil, nil)),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								_, err := testObserver.GetAreas()

								assert.Error(t, err)
							},
						},
					},
					{
						Name: "Successful refresh",
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:cmd/rt:app/rn:vinculum/ad:1", prime.CmdPD7Request, prime.ServiceName, &prime.Request{Cmd: prime.CmdGet, Param: &prime.RequestParam{Components: []string{prime.ComponentDevice, prime.ComponentThing, prime.ComponentRoom, prime.ComponentArea}}}).ReplyWith(
								fimpgo.NewObjectMessage(prime.EvtPD7Response, prime.ServiceName, &prime.Response{ParamRaw: map[string]json.RawMessage{
									prime.ComponentDevice: json.RawMessage(`[{"id":1}]`),
									prime.ComponentThing:  json.RawMessage(`[{"id":1}]`),
									prime.ComponentRoom:   json.RawMessage(`[{"id":1}]`),
									prime.ComponentArea:   json.RawMessage(`[{"id":1}]`),
								}}, nil, nil, nil)),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								err := testObserver.Refresh(false)

								assert.NoError(t, err)

								devices, err := testObserver.GetDevices()

								assert.NoError(t, err)
								assert.Len(t, devices, 1)

								things, err := testObserver.GetThings()

								assert.NoError(t, err)
								assert.Len(t, things, 1)

								rooms, err := testObserver.GetRooms()

								assert.NoError(t, err)
								assert.Len(t, rooms, 1)

								areas, err := testObserver.GetAreas()

								assert.NoError(t, err)
								assert.Len(t, areas, 1)
							},
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestObserver_AddOrEdit_NoComponentsAtStartup(t *testing.T) { //nolint:paralleltest
	var (
		testObserver     observer.Observer
		testEventManager event.Manager

		loggerHook = test.NewLocal(logrus.StandardLogger())
	)

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:  "Observer",
				Setup: setupObserverTest(&testObserver, &testEventManager, time.Millisecond),
				Nodes: []*suite.Node{
					suite.SleepNode(5 * time.Millisecond),
					{
						Name: "add new device",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentDevice,
							ParamRaw:  json.RawMessage(`{"id":1}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								time.Sleep(30 * time.Millisecond)
								assertNoPanicLogs(t, loggerHook)
							},
						},
					},
					{
						Name: "add new thing",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentThing,
							ParamRaw:  json.RawMessage(`{"id":1}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								time.Sleep(30 * time.Millisecond)
								assertNoPanicLogs(t, loggerHook)
							},
						},
					},
					{
						Name: "add new room",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentRoom,
							ParamRaw:  json.RawMessage(`{"id":1}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								time.Sleep(30 * time.Millisecond)
								assertNoPanicLogs(t, loggerHook)
							},
						},
					},
					{
						Name: "add new area",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentArea,
							ParamRaw:  json.RawMessage(`{"id":1}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								time.Sleep(30 * time.Millisecond)
								assertNoPanicLogs(t, loggerHook)
							},
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func setupObserverTest(testObserver *observer.Observer, testEventManager *event.Manager, primeClientTimeout time.Duration) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) (routing []*router.Routing, tasks []*task.Task, mocks []suite.Mock) {
		syncClient := fimpgo.NewSyncClient(mqtt)
		primeClient := prime.NewClient(syncClient, "testResource", primeClientTimeout)
		*testEventManager = event.NewManager()

		var err error

		*testObserver, err = observer.New(primeClient, *testEventManager, time.Hour, prime.ComponentDevice, prime.ComponentThing, prime.ComponentRoom, prime.ComponentArea)
		if err != nil {
			t.Fatalf("failed to create a new observer: %s", err)
		}

		return observer.RouteObserver(*testObserver), observer.TaskObserver(*testObserver, time.Minute), nil
	}
}

func assertNoPanicLogs(t *testing.T, hook *test.Hook) {
	defer hook.Reset()

	for _, entry := range hook.AllEntries() {
		assert.NotContains(t, entry.Message, "panic")
	}
}
