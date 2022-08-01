package observer_test

//func TestObserver(t *testing.T) {
//	var testObserver observer.Observer
//	var testEventManager event.Manager
//
//	s := &suite.Suite{
//		Cases: []*suite.Case{
//			{
//				Name: "Observer",
//				Setup: suite.BaseSetup(func(t *testing.T, mqtt *fimpgo.MqttTransport) (routing []*router.Routing, tasks []*task.Task, mocks []suite.Mock) {
//					syncClient := fimpgo.NewSyncClient(mqtt)
//					primeClient := prime.NewClient(syncClient, "testResource")
//					testEventManager = event.NewManager()
//
//					var err error
//
//					testObserver, err = observer.New(primeClient, testEventManager, time.Hour, prime.ComponentDevice, prime.ComponentThing, prime.ComponentRoom, prime.ComponentArea)
//					if err != nil {
//						t.Fatalf("failed to create a new observer: %s", err)
//					}
//
//					return observer.RouteObserver(testObserver), observer.TaskObserver(testObserver, time.Minute), nil
//				}),
//				Nodes: []*suite.Node{
//					{
//						Name: "Initialize observer on startup",
//						Expectations: []*suite.Expectation{
//							suite.ExpectObject("pt:j1/mt:cmd/rt:app/rn:vinculum/ad:1", prime.CmdPD7Request, prime.ServiceName, &prime.Request{Cmd: prime.CmdGet, Param: &prime.RequestParam{Components: []string{prime.ComponentDevice, prime.ComponentThing, prime.ComponentRoom, prime.ComponentArea}}}).ReplyWith(
//								fimpgo.NewObjectMessage(prime.EvtPD7Response, prime.ServiceName, &prime.Response{ParamRaw: map[string]json.RawMessage{
//									prime.ComponentDevice: json.RawMessage(`[{"id":1}]`),
//									prime.ComponentThing:  json.RawMessage(`[{"id":1}]`),
//									prime.ComponentRoom:   json.RawMessage(`[{"id":1}]`),
//									prime.ComponentArea:   json.RawMessage(`[{"id":1}]`),
//								}}, nil, nil, nil)),
//						},
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(10*time.Second, observer.WaitForRefresh(prime.ComponentDevice)))
//
//								devices, err := testObserver.GetDevices()
//
//								assert.NoError(t, err)
//								assert.Len(t, devices, 1)
//
//								things, err := testObserver.GetThings()
//
//								assert.NoError(t, err)
//								assert.Len(t, things, 1)
//
//								rooms, err := testObserver.GetRooms()
//
//								assert.NoError(t, err)
//								assert.Len(t, rooms, 1)
//
//								areas, err := testObserver.GetAreas()
//
//								assert.NoError(t, err)
//								assert.Len(t, areas, 1)
//							},
//						},
//					},
//					{
//						Name: "Added new device",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdAdd,
//							Component: prime.ComponentDevice,
//							ParamRaw:  json.RawMessage(`{"id":2}`),
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForDeviceChange()))
//
//								devices, err := testObserver.GetDevices()
//
//								assert.NoError(t, err)
//								assert.Len(t, devices, 2)
//							},
//						},
//					},
//					{
//						Name: "Add existing device",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdAdd,
//							Component: prime.ComponentDevice,
//							ParamRaw:  json.RawMessage(`{"id":2}`),
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForDeviceChange()))
//
//								devices, err := testObserver.GetDevices()
//
//								assert.NoError(t, err)
//								assert.Len(t, devices, 2)
//							},
//						},
//					},
//					{
//						Name: "Edit existing device",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdEdit,
//							Component: prime.ComponentDevice,
//							ParamRaw:  json.RawMessage(`{"id":2}`),
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForDeviceChange()))
//
//								devices, err := testObserver.GetDevices()
//
//								assert.NoError(t, err)
//								assert.Len(t, devices, 2)
//							},
//						},
//					},
//					{
//						Name: "Edit new device",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdEdit,
//							Component: prime.ComponentDevice,
//							ParamRaw:  json.RawMessage(`{"id":3}`),
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForDeviceChange()))
//
//								devices, err := testObserver.GetDevices()
//
//								assert.NoError(t, err)
//								assert.Len(t, devices, 3)
//							},
//						},
//					},
//					{
//						Name: "Delete device",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdDelete,
//							Component: prime.ComponentDevice,
//							ID:        3,
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForDeviceChange()))
//
//								devices, err := testObserver.GetDevices()
//
//								assert.NoError(t, err)
//								assert.Len(t, devices, 2)
//							},
//						},
//					},
//					{
//						Name: "Added new thing",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdAdd,
//							Component: prime.ComponentThing,
//							ParamRaw:  json.RawMessage(`{"id":2}`),
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForThingChange()))
//
//								things, err := testObserver.GetThings()
//
//								assert.NoError(t, err)
//								assert.Len(t, things, 2)
//							},
//						},
//					},
//					{
//						Name: "Add existing thing",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdAdd,
//							Component: prime.ComponentThing,
//							ParamRaw:  json.RawMessage(`{"id":2}`),
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForThingChange()))
//
//								things, err := testObserver.GetThings()
//
//								assert.NoError(t, err)
//								assert.Len(t, things, 2)
//							},
//						},
//					},
//					{
//						Name: "Edit existing thing",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdEdit,
//							Component: prime.ComponentThing,
//							ParamRaw:  json.RawMessage(`{"id":2}`),
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForThingChange()))
//
//								things, err := testObserver.GetThings()
//
//								assert.NoError(t, err)
//								assert.Len(t, things, 2)
//							},
//						},
//					},
//					{
//						Name: "Edit new thing",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdEdit,
//							Component: prime.ComponentThing,
//							ParamRaw:  json.RawMessage(`{"id":3}`),
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForThingChange()))
//
//								things, err := testObserver.GetThings()
//
//								assert.NoError(t, err)
//								assert.Len(t, things, 3)
//							},
//						},
//					},
//					{
//						Name: "Delete thing",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdDelete,
//							Component: prime.ComponentThing,
//							ID:        3,
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForThingChange()))
//
//								things, err := testObserver.GetThings()
//
//								assert.NoError(t, err)
//								assert.Len(t, things, 2)
//							},
//						},
//					},
//					{
//						Name: "Added new room",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdAdd,
//							Component: prime.ComponentRoom,
//							ParamRaw:  json.RawMessage(`{"id":2}`),
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForRoomChange()))
//
//								rooms, err := testObserver.GetRooms()
//
//								assert.NoError(t, err)
//								assert.Len(t, rooms, 2)
//							},
//						},
//					},
//					{
//						Name: "Add existing room",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdAdd,
//							Component: prime.ComponentRoom,
//							ParamRaw:  json.RawMessage(`{"id":2}`),
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForRoomChange()))
//
//								rooms, err := testObserver.GetRooms()
//
//								assert.NoError(t, err)
//								assert.Len(t, rooms, 2)
//							},
//						},
//					},
//					{
//						Name: "Edit existing room",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdEdit,
//							Component: prime.ComponentRoom,
//							ParamRaw:  json.RawMessage(`{"id":2}`),
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForRoomChange()))
//
//								rooms, err := testObserver.GetRooms()
//
//								assert.NoError(t, err)
//								assert.Len(t, rooms, 2)
//							},
//						},
//					},
//					{
//						Name: "Edit new room",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdEdit,
//							Component: prime.ComponentRoom,
//							ParamRaw:  json.RawMessage(`{"id":3}`),
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForRoomChange()))
//
//								rooms, err := testObserver.GetRooms()
//
//								assert.NoError(t, err)
//								assert.Len(t, rooms, 3)
//							},
//						},
//					},
//					{
//						Name: "Delete room",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdDelete,
//							Component: prime.ComponentRoom,
//							ID:        3,
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForRoomChange()))
//
//								rooms, err := testObserver.GetRooms()
//
//								assert.NoError(t, err)
//								assert.Len(t, rooms, 2)
//							},
//						},
//					},
//					{
//						Name: "Added new area",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdAdd,
//							Component: prime.ComponentArea,
//							ParamRaw:  json.RawMessage(`{"id":2}`),
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForAreaChange()))
//
//								areas, err := testObserver.GetAreas()
//
//								assert.NoError(t, err)
//								assert.Len(t, areas, 2)
//							},
//						},
//					},
//					{
//						Name: "Add existing area",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdAdd,
//							Component: prime.ComponentArea,
//							ParamRaw:  json.RawMessage(`{"id":2}`),
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForAreaChange()))
//
//								areas, err := testObserver.GetAreas()
//
//								assert.NoError(t, err)
//								assert.Len(t, areas, 2)
//							},
//						},
//					},
//					{
//						Name: "Edit existing area",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdEdit,
//							Component: prime.ComponentArea,
//							ParamRaw:  json.RawMessage(`{"id":2}`),
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForAreaChange()))
//
//								areas, err := testObserver.GetAreas()
//
//								assert.NoError(t, err)
//								assert.Len(t, areas, 2)
//							},
//						},
//					},
//					{
//						Name: "Edit new area",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdEdit,
//							Component: prime.ComponentArea,
//							ParamRaw:  json.RawMessage(`{"id":3}`),
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForAreaChange()))
//
//								areas, err := testObserver.GetAreas()
//
//								assert.NoError(t, err)
//								assert.Len(t, areas, 3)
//							},
//						},
//					},
//					{
//						Name: "Delete area",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdDelete,
//							Component: prime.ComponentArea,
//							ID:        3,
//						}),
//						Callbacks: []suite.Callback{
//							func(t *testing.T) {
//								assert.NotNil(t, <-testEventManager.WaitFor(time.Second, observer.WaitForAreaChange()))
//
//								areas, err := testObserver.GetAreas()
//
//								assert.NoError(t, err)
//								assert.Len(t, areas, 2)
//							},
//						},
//					},
//					{
//						Name: "Unobserved notification",
//						Command: suite.ObjectMessage(prime.NotifyTopic, prime.EvtPD7Notify, prime.ServiceName, &prime.Notify{
//							Cmd:       prime.CmdSet,
//							Component: prime.ComponentRoom,
//							ID:        1,
//						}),
//					},
//				},
//			},
//		},
//	}
//
//	s.Run(t)
//}
