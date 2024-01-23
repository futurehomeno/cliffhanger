package prime_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/futurehomeno/cliffhanger/prime"
	mockedprime "github.com/futurehomeno/cliffhanger/test/mocks/prime"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	m := mockedprime.NewSyncClient(t)

	m.On("SendReqRespFimp",
		"pt:j1/mt:cmd/rt:app/rn:vinculum/ad:1",
		"pt:j1/mt:rsp/rt:app/rn:test_application/ad:1",
		mock.Anything, mock.Anything, mock.Anything,
	).Return(nil, errors.New("test error"))

	client := prime.NewClient(m, "test_application", 5*time.Second)

	_, err := client.GetAll()

	assert.Error(t, err)

	m.AssertExpectations(t)
}

func TestNewCloudClient(t *testing.T) {
	t.Parallel()

	m := mockedprime.NewSyncClient(t)

	m.On("SendReqRespFimp",
		"test_site_uuid/pt:j1/mt:cmd/rt:app/rn:vinculum/ad:1",
		"test_site_uuid/pt:j1/mt:rsp/rt:cloud/rn:backend-service/ad:test_cloud_service",
		mock.Anything, mock.Anything, mock.Anything,
	).Return(nil, errors.New("test error"))

	client := prime.NewCloudClient(m, "test_cloud_service", "test_site_uuid", 5*time.Second)

	_, err := client.GetAll()

	assert.Error(t, err)

	m.AssertExpectations(t)
}

func TestClient(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name           string
		syncClientMock *mockedprime.SyncClient
		call           func(client prime.Client) (interface{}, error)
		want           interface{}
		wantErr        bool
	}{
		{
			name: "Successful get devices",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentDevice: json.RawMessage(`[{"id":1},{"id":2}]`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetDevices() },
			want:    prime.Devices{{ID: 1}, {ID: 2}},
			wantErr: false,
		},
		{
			name: "Empty get devices response",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetDevices() },
			want:    (prime.Devices)(nil),
			wantErr: false,
		},
		{
			name: "Invalid get devices payload",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentDevice: json.RawMessage(`[{"id":"1"},{"id":"2"}]`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetDevices() },
			want:    (prime.Devices)(nil),
			wantErr: true,
		},
		{
			name:           "Get devices request error",
			syncClientMock: mockSyncClientResponse(t, nil, errors.New("test")),
			call:           func(client prime.Client) (interface{}, error) { return client.GetDevices() },
			want:           (prime.Devices)(nil),
			wantErr:        true,
		},
		{
			name: "Successful get things",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentThing: json.RawMessage(`[{"id":1},{"id":2}]`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetThings() },
			want:    prime.Things{{ID: 1}, {ID: 2}},
			wantErr: false,
		},
		{
			name: "Empty get things response",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetThings() },
			want:    (prime.Things)(nil),
			wantErr: false,
		},
		{
			name: "Invalid get things payload",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentThing: json.RawMessage(`[{"id":"1"},{"id":"2"}]`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetThings() },
			want:    (prime.Things)(nil),
			wantErr: true,
		},
		{
			name:           "Get things request error",
			syncClientMock: mockSyncClientResponse(t, nil, errors.New("test")),
			call:           func(client prime.Client) (interface{}, error) { return client.GetThings() },
			want:           (prime.Things)(nil),
			wantErr:        true,
		},
		{
			name: "Successful get Rooms",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentRoom: json.RawMessage(`[{"id":1},{"id":2}]`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetRooms() },
			want:    prime.Rooms{{ID: 1}, {ID: 2}},
			wantErr: false,
		},
		{
			name: "Empty get Rooms response",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetRooms() },
			want:    (prime.Rooms)(nil),
			wantErr: false,
		},
		{
			name: "Invalid get Rooms payload",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentRoom: json.RawMessage(`[{"id":"1"},{"id":"2"}]`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetRooms() },
			want:    (prime.Rooms)(nil),
			wantErr: true,
		},
		{
			name:           "Get Rooms request error",
			syncClientMock: mockSyncClientResponse(t, nil, errors.New("test")),
			call:           func(client prime.Client) (interface{}, error) { return client.GetRooms() },
			want:           (prime.Rooms)(nil),
			wantErr:        true,
		},
		{
			name: "Successful get Areas",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentArea: json.RawMessage(`[{"id":1},{"id":2}]`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetAreas() },
			want:    prime.Areas{{ID: 1}, {ID: 2}},
			wantErr: false,
		},
		{
			name: "Empty get Areas response",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetAreas() },
			want:    (prime.Areas)(nil),
			wantErr: false,
		},
		{
			name: "Invalid get Areas payload",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentArea: json.RawMessage(`[{"id":"1"},{"id":"2"}]`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetAreas() },
			want:    (prime.Areas)(nil),
			wantErr: true,
		},
		{
			name:           "Get Areas request error",
			syncClientMock: mockSyncClientResponse(t, nil, errors.New("test")),
			call:           func(client prime.Client) (interface{}, error) { return client.GetAreas() },
			want:           (prime.Areas)(nil),
			wantErr:        true,
		},
		{
			name: "Successful get House",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentHouse: json.RawMessage(`{"mode":"Away"}`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetHouse() },
			want:    &prime.House{Mode: "Away"},
			wantErr: false,
		},
		{
			name: "Empty get House response",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetHouse() },
			want:    (*prime.House)(nil),
			wantErr: false,
		},
		{
			name: "Invalid get House payload",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentHouse: json.RawMessage(`{"mode":1}`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetHouse() },
			want:    (*prime.House)(nil),
			wantErr: true,
		},
		{
			name:           "Get House request error",
			syncClientMock: mockSyncClientResponse(t, nil, errors.New("test")),
			call:           func(client prime.Client) (interface{}, error) { return client.GetHouse() },
			want:           (*prime.House)(nil),
			wantErr:        true,
		},
		{
			name: "Successful get Hub",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentHub: json.RawMessage(`{"mode":{"current":"Away"}}`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetHub() },
			want:    &prime.Hub{Mode: prime.HubMode{Current: "Away"}},
			wantErr: false,
		},
		{
			name: "Empty get Hub response",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetHub() },
			want:    (*prime.Hub)(nil),
			wantErr: false,
		},
		{
			name: "Invalid get Hub payload",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentHub: json.RawMessage(`{"mode":1}`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetHub() },
			want:    (*prime.Hub)(nil),
			wantErr: true,
		},
		{
			name:           "Get Hub request error",
			syncClientMock: mockSyncClientResponse(t, nil, errors.New("test")),
			call:           func(client prime.Client) (interface{}, error) { return client.GetHub() },
			want:           (*prime.Hub)(nil),
			wantErr:        true,
		},
		{
			name: "Successful get Shortcuts",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentShortcut: json.RawMessage(`[{"id":1},{"id":2}]`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetShortcuts() },
			want:    prime.Shortcuts{{ID: 1}, {ID: 2}},
			wantErr: false,
		},
		{
			name: "Empty get Shortcuts response",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetShortcuts() },
			want:    (prime.Shortcuts)(nil),
			wantErr: false,
		},
		{
			name: "Invalid get Shortcuts payload",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentShortcut: json.RawMessage(`[{"id":"1"},{"id":"2"}]`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetShortcuts() },
			want:    (prime.Shortcuts)(nil),
			wantErr: true,
		},
		{
			name:           "Get Shortcuts request error",
			syncClientMock: mockSyncClientResponse(t, nil, errors.New("test")),
			call:           func(client prime.Client) (interface{}, error) { return client.GetShortcuts() },
			want:           (prime.Shortcuts)(nil),
			wantErr:        true,
		},
		{
			name: "Successful get modes",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentMode: json.RawMessage(`[{"id":"home"},{"id":"away"}]`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetModes() },
			want:    prime.Modes{{ID: "home"}, {ID: "away"}},
			wantErr: false,
		},
		{
			name: "Empty get modes response",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{},
				Success:  true,
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetModes() },
			want:    (prime.Modes)(nil),
			wantErr: false,
		},
		{
			name: "Invalid get modes payload",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentMode: json.RawMessage(`[{"id":1},{"id":2}]`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetModes() },
			want:    (prime.Modes)(nil),
			wantErr: true,
		},
		{
			name:           "Get modes request error",
			syncClientMock: mockSyncClientResponse(t, nil, errors.New("test")),
			call:           func(client prime.Client) (interface{}, error) { return client.GetModes() },
			want:           (prime.Modes)(nil),
			wantErr:        true,
		},
		{
			name: "Successful get Timers",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentTimer: json.RawMessage(`[{"id":1},{"id":2}]`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetTimers() },
			want:    prime.Timers{{ID: 1}, {ID: 2}},
			wantErr: false,
		},
		{
			name: "Empty get Timers response",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetTimers() },
			want:    (prime.Timers)(nil),
			wantErr: false,
		},
		{
			name: "Invalid get Timers payload",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentTimer: json.RawMessage(`[{"id":"1"},{"id":"2"}]`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetTimers() },
			want:    (prime.Timers)(nil),
			wantErr: true,
		},
		{
			name:           "Get Timers request error",
			syncClientMock: mockSyncClientResponse(t, nil, errors.New("test")),
			call:           func(client prime.Client) (interface{}, error) { return client.GetTimers() },
			want:           (prime.Timers)(nil),
			wantErr:        true,
		},
		{
			name: "Successful get services",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentService: json.RawMessage(`{}`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetVinculumServices() },
			want:    &prime.VinculumServices{},
			wantErr: false,
		},
		{
			name: "Empty get services response",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetVinculumServices() },
			want:    (*prime.VinculumServices)(nil),
			wantErr: false,
		},
		{
			name: "Invalid get services payload",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentService: json.RawMessage(`{"fireAlarm":1}`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetVinculumServices() },
			want:    (*prime.VinculumServices)(nil),
			wantErr: true,
		},
		{
			name:           "Get services request error",
			syncClientMock: mockSyncClientResponse(t, nil, errors.New("test")),
			call:           func(client prime.Client) (interface{}, error) { return client.GetVinculumServices() },
			want:           (*prime.VinculumServices)(nil),
			wantErr:        true,
		},
		{
			name: "Successful get State",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentState: json.RawMessage(`{}`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetState() },
			want:    &prime.State{},
			wantErr: false,
		},
		{
			name: "Empty get State response",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetState() },
			want:    (*prime.State)(nil),
			wantErr: false,
		},
		{
			name: "Invalid get State payload",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentState: json.RawMessage(`{"devices":1}`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetState() },
			want:    (*prime.State)(nil),
			wantErr: true,
		},
		{
			name:           "Get State request error",
			syncClientMock: mockSyncClientResponse(t, nil, errors.New("test")),
			call:           func(client prime.Client) (interface{}, error) { return client.GetState() },
			want:           (*prime.State)(nil),
			wantErr:        true,
		},
		{
			name: "Successful get all",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				ParamRaw: map[string]json.RawMessage{
					prime.ComponentDevice: json.RawMessage(`[{"id":1},{"id":2}]`),
				},
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.GetAll() },
			want:    &prime.ComponentSet{Devices: prime.Devices{{ID: 1}, {ID: 2}}},
			wantErr: false,
		},
		{
			name:           "Failed get all",
			syncClientMock: mockSyncClientResponse(t, nil, errors.New("test")),
			call:           func(client prime.Client) (interface{}, error) { return client.GetAll() },
			want:           (*prime.ComponentSet)(nil),
			wantErr:        true,
		},
		{
			name:           "Get unsupported components",
			syncClientMock: mockedprime.NewSyncClient(t),
			call:           func(client prime.Client) (interface{}, error) { return client.GetComponents("unknown") },
			want:           (*prime.ComponentSet)(nil),
			wantErr:        true,
		},
		{
			name: "Successful run shortcut",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				Success: true,
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.RunShortcut(1) },
			want:    &prime.Response{Success: true},
			wantErr: false,
		},
		{
			name:           "Failed run shortcut",
			syncClientMock: mockSyncClientResponse(t, nil, errors.New("test")),
			call:           func(client prime.Client) (interface{}, error) { return client.RunShortcut(1) },
			want:           (*prime.Response)(nil),
			wantErr:        true,
		},
		{
			name: "Successful change mode",
			syncClientMock: mockSyncClientResponse(t, &prime.Response{
				Success: true,
			}, nil),
			call:    func(client prime.Client) (interface{}, error) { return client.ChangeMode("away") },
			want:    &prime.Response{Success: true},
			wantErr: false,
		},
		{
			name:           "Failed change mode",
			syncClientMock: mockSyncClientResponse(t, nil, errors.New("test")),
			call:           func(client prime.Client) (interface{}, error) { return client.ChangeMode("away") },
			want:           (*prime.Response)(nil),
			wantErr:        true,
		},
		{
			name:           "Get request invalid response payload",
			syncClientMock: mockSyncClientResponse(t, json.RawMessage(`{"cmd":true}`), nil),
			call:           func(client prime.Client) (interface{}, error) { return client.GetDevices() },
			want:           (prime.Devices)(nil),
			wantErr:        true,
		},
		{
			name:           "Set request invalid response payload",
			syncClientMock: mockSyncClientResponse(t, json.RawMessage(`{"cmd":true}`), nil),
			call:           func(client prime.Client) (interface{}, error) { return client.ChangeMode("home") },
			want:           (*prime.Response)(nil),
			wantErr:        true,
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := prime.NewClient(tc.syncClientMock, "test", 5*time.Second)

			got, err := tc.call(client)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.want, got)

			tc.syncClientMock.AssertExpectations(t)
		})
	}
}

func mockSyncClientResponse(t *testing.T, response interface{}, err error) *mockedprime.SyncClient {
	t.Helper()

	var responseMsg *fimpgo.FimpMessage

	if err == nil {
		b, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("error while marshaling response: %s", err)
		}

		responseMsg = fimpgo.NewObjectMessage(prime.EvtPD7Response, "vinculum", response, nil, nil, nil)
		responseMsg.Source = "vinculum"
		responseMsg.ValueObj = b
	}

	m := mockedprime.NewSyncClient(t)

	m.On("SendReqRespFimp", "pt:j1/mt:cmd/rt:app/rn:vinculum/ad:1", "pt:j1/mt:rsp/rt:app/rn:test/ad:1", mock.Anything, mock.Anything, mock.Anything).
		Return(responseMsg, err)

	return m
}
