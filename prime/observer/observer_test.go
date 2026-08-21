package observer_test

import (
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
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
							suite.ExpectObject(
								"pt:j1/mt:cmd/rt:app/rn:vinculum/ad:1", prime.CmdPD7Request, fimptype.VinculumService,
								&prime.Request{Cmd: prime.CmdGet, Param: &prime.RequestParam{Components: []string{prime.ComponentDevice, prime.ComponentThing, prime.ComponentRoom, prime.ComponentArea}}}).ReplyWith(
								fimpgo.NewObjectMessage(prime.EvtPD7Response, fimptype.VinculumService, &prime.Response{ParamRaw: map[string]json.RawMessage{
									prime.ComponentDevice: json.RawMessage(`[{"id":1}]`),
									prime.ComponentThing:  json.RawMessage(`[{"id":1}]`),
									prime.ComponentRoom:   json.RawMessage(`[{"id":1}]`),
									prime.ComponentArea:   json.RawMessage(`[{"id":1}]`),
								}}, nil, nil, nil)),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
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
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentDevice,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForDeviceChange()))

								devices, err := testObserver.GetDevices()

								assert.NoError(t, err)
								assert.Len(t, devices, 2)
							},
						},
					},
					{
						Name: "add existing device",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentDevice,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForDeviceChange()))

								devices, err := testObserver.GetDevices()

								assert.NoError(t, err)
								assert.Len(t, devices, 2)
							},
						},
					},
					{
						Name: "Edit existing device",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentDevice,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForDeviceChange()))

								devices, err := testObserver.GetDevices()

								assert.NoError(t, err)
								assert.Len(t, devices, 2)
							},
						},
					},
					{
						Name: "Edit new device",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentDevice,
							ParamRaw:  json.RawMessage(`{"id":3}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForDeviceChange()))

								devices, err := testObserver.GetDevices()

								assert.NoError(t, err)
								assert.Len(t, devices, 3)
							},
						},
					},
					{
						Name: "Delete device",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdDelete,
							Component: prime.ComponentDevice,
							ID:        3,
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForDeviceChange()))

								devices, err := testObserver.GetDevices()

								assert.NoError(t, err)
								assert.Len(t, devices, 2)
							},
						},
					},
					{
						Name: "add new thing",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentThing,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForThingChange()))

								things, err := testObserver.GetThings()

								assert.NoError(t, err)
								assert.Len(t, things, 2)
							},
						},
					},
					{
						Name: "add existing thing",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentThing,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForThingChange()))

								things, err := testObserver.GetThings()

								assert.NoError(t, err)
								assert.Len(t, things, 2)
							},
						},
					},
					{
						Name: "Edit existing thing",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentThing,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForThingChange()))

								things, err := testObserver.GetThings()

								assert.NoError(t, err)
								assert.Len(t, things, 2)
							},
						},
					},
					{
						Name: "Edit new thing",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentThing,
							ParamRaw:  json.RawMessage(`{"id":3}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForThingChange()))

								things, err := testObserver.GetThings()

								assert.NoError(t, err)
								assert.Len(t, things, 3)
							},
						},
					},
					{
						Name: "Delete thing",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdDelete,
							Component: prime.ComponentThing,
							ID:        3,
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForThingChange()))

								things, err := testObserver.GetThings()

								assert.NoError(t, err)
								assert.Len(t, things, 2)
							},
						},
					},
					{
						Name: "add new room",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentRoom,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForRoomChange()))

								rooms, err := testObserver.GetRooms()

								assert.NoError(t, err)
								assert.Len(t, rooms, 2)
							},
						},
					},
					{
						Name: "add existing room",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentRoom,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForRoomChange()))

								rooms, err := testObserver.GetRooms()

								assert.NoError(t, err)
								assert.Len(t, rooms, 2)
							},
						},
					},
					{
						Name: "Edit existing room",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentRoom,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForRoomChange()))

								rooms, err := testObserver.GetRooms()

								assert.NoError(t, err)
								assert.Len(t, rooms, 2)
							},
						},
					},
					{
						Name: "Edit new room",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentRoom,
							ParamRaw:  json.RawMessage(`{"id":3}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForRoomChange()))

								rooms, err := testObserver.GetRooms()

								assert.NoError(t, err)
								assert.Len(t, rooms, 3)
							},
						},
					},
					{
						Name: "Delete room",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdDelete,
							Component: prime.ComponentRoom,
							ID:        3,
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForRoomChange()))

								rooms, err := testObserver.GetRooms()

								assert.NoError(t, err)
								assert.Len(t, rooms, 2)
							},
						},
					},
					{
						Name: "Added new area",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentArea,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForAreaChange()))

								areas, err := testObserver.GetAreas()

								assert.NoError(t, err)
								assert.Len(t, areas, 2)
							},
						},
					},
					{
						Name: "add existing area",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentArea,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForAreaChange()))

								areas, err := testObserver.GetAreas()

								assert.NoError(t, err)
								assert.Len(t, areas, 2)
							},
						},
					},
					{
						Name: "Edit existing area",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentArea,
							ParamRaw:  json.RawMessage(`{"id":2}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForAreaChange()))

								areas, err := testObserver.GetAreas()

								assert.NoError(t, err)
								assert.Len(t, areas, 2)
							},
						},
					},
					{
						Name: "Edit new area",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentArea,
							ParamRaw:  json.RawMessage(`{"id":3}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForAreaChange()))

								areas, err := testObserver.GetAreas()

								assert.NoError(t, err)
								assert.Len(t, areas, 3)
							},
						},
					},
					{
						Name: "Delete area",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdDelete,
							Component: prime.ComponentArea,
							ID:        3,
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForAreaChange()))

								areas, err := testObserver.GetAreas()

								assert.NoError(t, err)
								assert.Len(t, areas, 2)
							},
						},
					},
					{
						Name:    "Corrupted notification",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, json.RawMessage(`{"cmd":1}`)),
					},
					{
						Name: "Unobserved notification",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdSet,
							Component: prime.ComponentState,
							ID:        1,
						}),
					},
					{
						Name: "Failed add new device",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentDevice,
							ParamRaw:  json.RawMessage(`{"id":"2"}`),
						}),
					},
					{
						Name: "Failed edit device",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentDevice,
							ParamRaw:  json.RawMessage(`{"id":"2"}`),
						}),
					},
					{
						Name: "Failed delete device",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdDelete,
							Component: prime.ComponentDevice,
							ID:        "A",
						}),
					},
					{
						Name: "Failed add new thing",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentThing,
							ParamRaw:  json.RawMessage(`{"id":"2"}`),
						}),
					},
					{
						Name: "Failed edit thing",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentThing,
							ParamRaw:  json.RawMessage(`{"id":"2"}`),
						}),
					},
					{
						Name: "Failed delete thing",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdDelete,
							Component: prime.ComponentThing,
							ID:        "a",
						}),
					},
					{
						Name: "Failed add new room",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentRoom,
							ParamRaw:  json.RawMessage(`{"id":"2"}`),
						}),
					},
					{
						Name: "Failed edit room",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentRoom,
							ParamRaw:  json.RawMessage(`{"id":"2"}`),
						}),
					},
					{
						Name: "Failed delete room",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdDelete,
							Component: prime.ComponentRoom,
							ID:        "A",
						}),
					},
					{
						Name: "Failed add new area",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentArea,
							ParamRaw:  json.RawMessage(`{"id":"2"}`),
						}),
					},
					{
						Name: "Failed edit area",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdEdit,
							Component: prime.ComponentArea,
							ParamRaw:  json.RawMessage(`{"id":"2"}`),
						}),
					},
					{
						Name: "Failed delete area",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdDelete,
							Component: prime.ComponentArea,
							ID:        "A",
						}),
					},
					suite.SleepNode(50 * time.Millisecond), // sleeping to allow observer to process all incoming messages
					{
						Name: "Failed lazy load on getting devices",
						Expectations: []*suite.Expectation{
							suite.ExpectObject(
								"pt:j1/mt:cmd/rt:app/rn:vinculum/ad:1", prime.CmdPD7Request, fimptype.VinculumService,
								&prime.Request{Cmd: prime.CmdGet,
									Param: &prime.RequestParam{Components: []string{prime.ComponentDevice, prime.ComponentThing, prime.ComponentRoom, prime.ComponentArea}}}).ReplyWith(
								fimpgo.NewObjectMessage(prime.EvtPD7Response, fimptype.VinculumService, json.RawMessage(`{"cmd":1}`), nil, nil, nil)),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								_, err := testObserver.GetDevices()

								assert.Error(t, err)
							},
						},
					},
					{
						Name: "Failed lazy load on getting things",
						Expectations: []*suite.Expectation{
							suite.ExpectObject(
								"pt:j1/mt:cmd/rt:app/rn:vinculum/ad:1", prime.CmdPD7Request, fimptype.VinculumService,
								&prime.Request{Cmd: prime.CmdGet,
									Param: &prime.RequestParam{Components: []string{prime.ComponentDevice, prime.ComponentThing, prime.ComponentRoom, prime.ComponentArea}}}).ReplyWith(
								fimpgo.NewObjectMessage(prime.EvtPD7Response, fimptype.VinculumService, json.RawMessage(`{"cmd":1}`), nil, nil, nil)),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								_, err := testObserver.GetThings()

								assert.Error(t, err)
							},
						},
					},
					{
						Name: "Failed lazy load on getting rooms",
						Expectations: []*suite.Expectation{
							suite.ExpectObject(
								"pt:j1/mt:cmd/rt:app/rn:vinculum/ad:1", prime.CmdPD7Request, fimptype.VinculumService,
								&prime.Request{Cmd: prime.CmdGet,
									Param: &prime.RequestParam{Components: []string{prime.ComponentDevice, prime.ComponentThing, prime.ComponentRoom, prime.ComponentArea}}}).ReplyWith(
								fimpgo.NewObjectMessage(prime.EvtPD7Response, fimptype.VinculumService, json.RawMessage(`{"cmd":1}`), nil, nil, nil)),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								_, err := testObserver.GetRooms()

								assert.Error(t, err)
							},
						},
					},
					{
						Name: "Failed lazy load on getting areas",
						Expectations: []*suite.Expectation{
							suite.ExpectObject(
								"pt:j1/mt:cmd/rt:app/rn:vinculum/ad:1", prime.CmdPD7Request, fimptype.VinculumService,
								&prime.Request{Cmd: prime.CmdGet, Param: &prime.RequestParam{
									Components: []string{prime.ComponentDevice, prime.ComponentThing, prime.ComponentRoom, prime.ComponentArea}}}).ReplyWith(
								fimpgo.NewObjectMessage(prime.EvtPD7Response, fimptype.VinculumService, json.RawMessage(`{"cmd":1}`), nil, nil, nil)),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								_, err := testObserver.GetAreas()

								assert.Error(t, err)
							},
						},
					},
					{
						Name: "Successful refresh",
						Expectations: []*suite.Expectation{
							suite.ExpectObject(
								"pt:j1/mt:cmd/rt:app/rn:vinculum/ad:1", prime.CmdPD7Request, fimptype.VinculumService,
								&prime.Request{Cmd: prime.CmdGet, Param: &prime.RequestParam{Components: []string{prime.ComponentDevice, prime.ComponentThing, prime.ComponentRoom, prime.ComponentArea}}}).ReplyWith(
								fimpgo.NewObjectMessage(prime.EvtPD7Response, fimptype.VinculumService, &prime.Response{ParamRaw: map[string]json.RawMessage{
									prime.ComponentDevice: json.RawMessage(`[{"id":1}]`),
									prime.ComponentThing:  json.RawMessage(`[{"id":1}]`),
									prime.ComponentRoom:   json.RawMessage(`[{"id":1}]`),
									prime.ComponentArea:   json.RawMessage(`[{"id":1}]`),
								}}, nil, nil, nil)),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
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
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentDevice,
							ParamRaw:  json.RawMessage(`{"id":1}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								time.Sleep(30 * time.Millisecond)
								assertNoPanicLogs(t, loggerHook)
							},
						},
					},
					{
						Name: "add new thing",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentThing,
							ParamRaw:  json.RawMessage(`{"id":1}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								time.Sleep(30 * time.Millisecond)
								assertNoPanicLogs(t, loggerHook)
							},
						},
					},
					{
						Name: "add new room",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentRoom,
							ParamRaw:  json.RawMessage(`{"id":1}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
								time.Sleep(30 * time.Millisecond)
								assertNoPanicLogs(t, loggerHook)
							},
						},
					},
					{
						Name: "add new area",
						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, fimptype.VinculumService, &prime.Notify{
							Cmd:       prime.CmdAdd,
							Component: prime.ComponentArea,
							ParamRaw:  json.RawMessage(`{"id":1}`),
						}),
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()
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
		t.Helper()
		syncClient := fimpgo.NewSyncClient(mqtt)
		primeClient := prime.NewClient(syncClient, "testResource", primeClientTimeout)
		*testEventManager = event.NewManager()

		var err error

		*testObserver, err = observer.New(primeClient, *testEventManager, time.Hour, prime.ComponentDevice, prime.ComponentThing, prime.ComponentRoom, prime.ComponentArea)
		if err != nil {
			t.Fatalf("failed to create a new observer: %s", err)
		}

		return observer.Route(*testObserver), observer.Task(*testObserver, time.Minute), nil
	}
}

func assertNoPanicLogs(t *testing.T, hook *test.Hook) {
	t.Helper()
	defer hook.Reset()

	for _, entry := range hook.AllEntries() {
		assert.NotContains(t, entry.Message, "panic")
	}
}

// TestObserver_ConcurrentGettersDoNotSerializeOnOneFetch pins that readers do not queue behind a
// slow request to vinculum. The getters used to take the write lock and make the request under it,
// so one stalled call blocked every other reader and the whole notification stream for its
// duration; they also made one request each rather than sharing one.
func TestObserver_ConcurrentGettersDoNotSerializeOnOneFetch(t *testing.T) {
	t.Parallel()

	client := &slowClient{}

	o, err := observer.New(client, event.NewManager(), time.Hour, prime.ComponentDevice)
	require.NoError(t, err)

	const readers = 8

	var wg sync.WaitGroup

	wg.Add(readers)

	start := time.Now()

	for range readers {
		go func() {
			defer wg.Done()

			devices, err := o.GetDevices()
			assert.NoError(t, err)
			assert.Len(t, devices, 1)
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(1), client.calls.Load(), "concurrent stale readers must share one refresh")
	assert.Less(t, time.Since(start), 1500*time.Millisecond, "readers must not serialize behind one another")
}

// slowClient stands in for a vinculum that answers slowly. Only GetComponents is ever called.
type slowClient struct {
	prime.Client

	calls atomic.Int32
}

func (c *slowClient) GetComponents(...string) (*prime.ComponentSet, error) {
	c.calls.Add(1)

	time.Sleep(200 * time.Millisecond)

	return &prime.ComponentSet{Devices: prime.Devices{{ID: 1}}}, nil
}

// blockingClient blocks its second GetComponents until released, so a notification can be applied
// while a refresh is in flight.
type blockingClient struct {
	prime.Client

	calls   atomic.Int32
	entered chan struct{}
	release chan struct{}
}

func (c *blockingClient) GetComponents(...string) (*prime.ComponentSet, error) {
	if c.calls.Add(1) == 2 {
		c.entered <- struct{}{}
		<-c.release
	}

	return &prime.ComponentSet{Devices: prime.Devices{{ID: 1}}}, nil
}

// TestObserver_RefreshKeepsUpdatesArrivingDuringFetch pins that a notification applied while the
// refresh is fetching survives. The fetch runs outside the lock, so installing its snapshot
// unconditionally would silently undo every update that landed in the meantime - the snapshot was
// requested before them.
func TestObserver_RefreshKeepsUpdatesArrivingDuringFetch(t *testing.T) {
	t.Parallel()

	client := &blockingClient{entered: make(chan struct{}), release: make(chan struct{})}

	o, err := observer.New(client, event.NewManager(), time.Hour, prime.ComponentDevice)
	require.NoError(t, err)

	// Seeds the cache so the notification below has a set to apply to.
	require.NoError(t, o.Refresh(true))

	done := make(chan error, 1)

	go func() { done <- o.Refresh(true) }()

	<-client.entered

	require.NoError(t, o.Update(&prime.Notify{
		Cmd:       prime.CmdAdd,
		Component: prime.ComponentDevice,
		ParamRaw:  json.RawMessage(`{"id":2}`),
	}))

	close(client.release)
	require.NoError(t, <-done)

	devices, err := o.GetDevices()
	require.NoError(t, err)
	assert.Len(t, devices, 2, "the device added during the fetch must not be discarded")
}

// failingClient stands in for an unreachable vinculum, answering slowly as a timing out request
// would.
type failingClient struct {
	prime.Client

	calls atomic.Int32
}

func (c *failingClient) GetComponents(...string) (*prime.ComponentSet, error) {
	c.calls.Add(1)

	time.Sleep(200 * time.Millisecond)

	return nil, errors.New("vinculum is down")
}

// TestObserver_ConcurrentGettersShareAFailedRefresh pins that a failed fetch is shared with the
// readers waiting behind it, the same way a successful one is. Without it every waiter would find
// the cache still stale and issue its own blocking retry, so an outage cost one request and one
// timeout per reader. Readers arriving afterwards must still retry.
func TestObserver_ConcurrentGettersShareAFailedRefresh(t *testing.T) {
	t.Parallel()

	client := &failingClient{}

	o, err := observer.New(client, event.NewManager(), time.Hour, prime.ComponentDevice)
	require.NoError(t, err)

	const readers = 8

	var wg sync.WaitGroup

	wg.Add(readers)

	for range readers {
		go func() {
			defer wg.Done()

			_, err := o.GetDevices()
			assert.Error(t, err)
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(1), client.calls.Load(), "concurrent stale readers must share one failed refresh")

	_, err = o.GetDevices()
	require.Error(t, err)
	assert.Equal(t, int32(2), client.calls.Load(), "a reader arriving after the failure must retry")
}
