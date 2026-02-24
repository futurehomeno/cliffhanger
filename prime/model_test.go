package prime_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/prime"
)

func TestResponse_GetAll(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name     string
		response *prime.Response
		want     *prime.ComponentSet
		wantErr  bool
	}{
		{
			name: "complete response",
			response: &prime.Response{ParamRaw: map[string]json.RawMessage{
				prime.ComponentDevice:   json.RawMessage(`[{"id":1},{"id":2}]`),
				prime.ComponentThing:    json.RawMessage(`[{"id":1},{"id":2}]`),
				prime.ComponentRoom:     json.RawMessage(`[{"id":1},{"id":2}]`),
				prime.ComponentArea:     json.RawMessage(`[{"id":1},{"id":2}]`),
				prime.ComponentHouse:    json.RawMessage(`{}`),
				prime.ComponentHub:      json.RawMessage(`{}`),
				prime.ComponentShortcut: json.RawMessage(`[{"id":1},{"id":2}]`),
				prime.ComponentMode:     json.RawMessage(`[{"id":"away"},{"id":"home"}]`),
				prime.ComponentTimer:    json.RawMessage(`[{"id":1},{"id":2}]`),
				prime.ComponentService:  json.RawMessage(`{}`),
				prime.ComponentState:    json.RawMessage(`{}`),
			}},
			want: &prime.ComponentSet{
				Devices:   prime.Devices{{ID: 1}, {ID: 2}},
				Things:    prime.Things{{ID: 1}, {ID: 2}},
				Rooms:     prime.Rooms{{ID: 1}, {ID: 2}},
				Areas:     prime.Areas{{ID: 1}, {ID: 2}},
				House:     &prime.House{},
				Hub:       &prime.Hub{},
				Shortcuts: prime.Shortcuts{{ID: 1}, {ID: 2}},
				Modes:     prime.Modes{{ID: "away"}, {ID: "home"}},
				Timers:    prime.Timers{{ID: 1}, {ID: 2}},
				Services:  &prime.VinculumServices{},
				State:     &prime.State{},
			},
			wantErr: false,
		},
		{
			name:     "empty response",
			response: &prime.Response{ParamRaw: nil},
			want:     &prime.ComponentSet{},
			wantErr:  false,
		},
		{
			name: "invalid response - devices",
			response: &prime.Response{ParamRaw: map[string]json.RawMessage{
				prime.ComponentDevice: json.RawMessage(`[{"id":"1"},{"id":"2"}]`),
			}},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid response - things",
			response: &prime.Response{ParamRaw: map[string]json.RawMessage{
				prime.ComponentThing: json.RawMessage(`[{"id":"1"},{"id":"2"}]`),
			}},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid response - rooms",
			response: &prime.Response{ParamRaw: map[string]json.RawMessage{
				prime.ComponentRoom: json.RawMessage(`[{"id":"1"},{"id":"2"}]`),
			}},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid response - areas",
			response: &prime.Response{ParamRaw: map[string]json.RawMessage{
				prime.ComponentArea: json.RawMessage(`[{"id":"1"},{"id":"2"}]`),
			}},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid response - house",
			response: &prime.Response{ParamRaw: map[string]json.RawMessage{
				prime.ComponentHouse: json.RawMessage(`{"mode":1}`),
			}},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid response - hub",
			response: &prime.Response{ParamRaw: map[string]json.RawMessage{
				prime.ComponentHub: json.RawMessage(`{"mode":1}`),
			}},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid response - shortcuts",
			response: &prime.Response{ParamRaw: map[string]json.RawMessage{
				prime.ComponentShortcut: json.RawMessage(`[{"id":"1"},{"id":"2"}]`),
			}},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid response - modes",
			response: &prime.Response{ParamRaw: map[string]json.RawMessage{
				prime.ComponentMode: json.RawMessage(`[{"id":1},{"id":2}]`),
			}},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid response - timers",
			response: &prime.Response{ParamRaw: map[string]json.RawMessage{
				prime.ComponentTimer: json.RawMessage(`[{"id":"1"},{"id":"2"}]`),
			}},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid response - services",
			response: &prime.Response{ParamRaw: map[string]json.RawMessage{
				prime.ComponentService: json.RawMessage(`{"fireAlarm":1}`),
			}},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid response - state",
			response: &prime.Response{ParamRaw: map[string]json.RawMessage{
				prime.ComponentState: json.RawMessage(`{"devices":1}`),
			}},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.response.GetAll()

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestNotify(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name    string
		notify  *prime.Notify
		call    func(n *prime.Notify) (any, error)
		want    any
		wantErr bool
	}{
		{
			name:    "notify with ID - float",
			notify:  &prime.Notify{ID: float64(1)},
			call:    func(n *prime.Notify) (any, error) { return n.ParseIntegerID() },
			want:    1,
			wantErr: false,
		},
		{
			name:    "notify with ID - int",
			notify:  &prime.Notify{ID: 1},
			call:    func(n *prime.Notify) (any, error) { return n.ParseIntegerID() },
			want:    1,
			wantErr: false,
		},
		{
			name:    "notify with ID - string",
			notify:  &prime.Notify{ID: "1"},
			call:    func(n *prime.Notify) (any, error) { return n.ParseIntegerID() },
			want:    1,
			wantErr: false,
		},
		{
			name:    "notify with no ID",
			notify:  &prime.Notify{},
			call:    func(n *prime.Notify) (any, error) { return n.ParseIntegerID() },
			want:    0,
			wantErr: true,
		},
		{
			name:    "notify with unknown type ID",
			notify:  &prime.Notify{ID: true},
			call:    func(n *prime.Notify) (any, error) { return n.ParseIntegerID() },
			want:    0,
			wantErr: true,
		},
		{
			name:    "notify with Device",
			notify:  &prime.Notify{Component: prime.ComponentDevice, ParamRaw: json.RawMessage(`{"id":1}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetDevice() },
			want:    &prime.Device{ID: 1},
			wantErr: false,
		},
		{
			name:    "notify with without Device",
			notify:  &prime.Notify{Component: prime.ComponentState},
			call:    func(n *prime.Notify) (any, error) { return n.GetDevice() },
			want:    (*prime.Device)(nil),
			wantErr: false,
		},
		{
			name:    "notify with corrupted Device",
			notify:  &prime.Notify{Component: prime.ComponentDevice, ParamRaw: json.RawMessage(`{"id":"1"}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetDevice() },
			want:    (*prime.Device)(nil),
			wantErr: true,
		},
		{
			name:    "notify with Thing",
			notify:  &prime.Notify{Component: prime.ComponentThing, ParamRaw: json.RawMessage(`{"id":1}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetThing() },
			want:    &prime.Thing{ID: 1},
			wantErr: false,
		},
		{
			name:    "notify with without Thing",
			notify:  &prime.Notify{Component: prime.ComponentState},
			call:    func(n *prime.Notify) (any, error) { return n.GetThing() },
			want:    (*prime.Thing)(nil),
			wantErr: false,
		},
		{
			name:    "notify with corrupted Thing",
			notify:  &prime.Notify{Component: prime.ComponentThing, ParamRaw: json.RawMessage(`{"id":"1"}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetThing() },
			want:    (*prime.Thing)(nil),
			wantErr: true,
		},
		{
			name:    "notify with Room",
			notify:  &prime.Notify{Component: prime.ComponentRoom, ParamRaw: json.RawMessage(`{"id":1}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetRoom() },
			want:    &prime.Room{ID: 1},
			wantErr: false,
		},
		{
			name:    "notify with without Room",
			notify:  &prime.Notify{Component: prime.ComponentState},
			call:    func(n *prime.Notify) (any, error) { return n.GetRoom() },
			want:    (*prime.Room)(nil),
			wantErr: false,
		},
		{
			name:    "notify with corrupted Room",
			notify:  &prime.Notify{Component: prime.ComponentRoom, ParamRaw: json.RawMessage(`{"id":"1"}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetRoom() },
			want:    (*prime.Room)(nil),
			wantErr: true,
		},
		{
			name:    "notify with Area",
			notify:  &prime.Notify{Component: prime.ComponentArea, ParamRaw: json.RawMessage(`{"id":1}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetArea() },
			want:    &prime.Area{ID: 1},
			wantErr: false,
		},
		{
			name:    "notify with without Area",
			notify:  &prime.Notify{Component: prime.ComponentState},
			call:    func(n *prime.Notify) (any, error) { return n.GetArea() },
			want:    (*prime.Area)(nil),
			wantErr: false,
		},
		{
			name:    "notify with corrupted Area",
			notify:  &prime.Notify{Component: prime.ComponentArea, ParamRaw: json.RawMessage(`{"id":"1"}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetArea() },
			want:    (*prime.Area)(nil),
			wantErr: true,
		},

		{
			name:    "notify with House",
			notify:  &prime.Notify{Component: prime.ComponentHouse, ParamRaw: json.RawMessage(`{}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetHouse() },
			want:    &prime.House{},
			wantErr: false,
		},
		{
			name:    "notify with without House",
			notify:  &prime.Notify{Component: prime.ComponentState},
			call:    func(n *prime.Notify) (any, error) { return n.GetHouse() },
			want:    (*prime.House)(nil),
			wantErr: false,
		},
		{
			name:    "notify with corrupted House",
			notify:  &prime.Notify{Component: prime.ComponentHouse, ParamRaw: json.RawMessage(`{"mode":1}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetHouse() },
			want:    (*prime.House)(nil),
			wantErr: true,
		},
		{
			name:    "notify with Hub",
			notify:  &prime.Notify{Component: prime.ComponentHub, ParamRaw: json.RawMessage(`{}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetHub() },
			want:    &prime.Hub{},
			wantErr: false,
		},
		{
			name:    "notify with without Hub",
			notify:  &prime.Notify{Component: prime.ComponentState},
			call:    func(n *prime.Notify) (any, error) { return n.GetHub() },
			want:    (*prime.Hub)(nil),
			wantErr: false,
		},
		{
			name:    "notify with corrupted Hub",
			notify:  &prime.Notify{Component: prime.ComponentHub, ParamRaw: json.RawMessage(`{"mode":"1"}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetHub() },
			want:    (*prime.Hub)(nil),
			wantErr: true,
		},
		{
			name:    "notify with hub mode",
			notify:  &prime.Notify{ID: "mode", Component: prime.ComponentHub, ParamRaw: json.RawMessage(`{"current":"home"}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetHubMode() },
			want:    &prime.HubMode{Current: "home"},
			wantErr: false,
		},
		{
			name:    "notify without hub mode",
			notify:  &prime.Notify{Component: prime.ComponentState},
			call:    func(n *prime.Notify) (any, error) { return n.GetHubMode() },
			want:    (*prime.HubMode)(nil),
			wantErr: false,
		},
		{
			name:    "notify with corrupted hub mode",
			notify:  &prime.Notify{ID: "mode", Component: prime.ComponentHub, ParamRaw: json.RawMessage(`{"current":1}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetHubMode() },
			want:    (*prime.HubMode)(nil),
			wantErr: true,
		},
		{
			name:    "notify with Shortcut",
			notify:  &prime.Notify{Component: prime.ComponentShortcut, ParamRaw: json.RawMessage(`{"id":1}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetShortcut() },
			want:    &prime.Shortcut{ID: 1},
			wantErr: false,
		},
		{
			name:    "notify with without Shortcut",
			notify:  &prime.Notify{Component: prime.ComponentState},
			call:    func(n *prime.Notify) (any, error) { return n.GetShortcut() },
			want:    (*prime.Shortcut)(nil),
			wantErr: false,
		},
		{
			name:    "notify with corrupted Shortcut",
			notify:  &prime.Notify{Component: prime.ComponentShortcut, ParamRaw: json.RawMessage(`{"id":"1"}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetShortcut() },
			want:    (*prime.Shortcut)(nil),
			wantErr: true,
		},
		{
			name:    "notify with Mode",
			notify:  &prime.Notify{Component: prime.ComponentMode, ParamRaw: json.RawMessage(`{"id":"home"}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetMode() },
			want:    &prime.Mode{ID: "home"},
			wantErr: false,
		},
		{
			name:    "notify with without Mode",
			notify:  &prime.Notify{Component: prime.ComponentState},
			call:    func(n *prime.Notify) (any, error) { return n.GetMode() },
			want:    (*prime.Mode)(nil),
			wantErr: false,
		},
		{
			name:    "notify with corrupted Mode",
			notify:  &prime.Notify{Component: prime.ComponentMode, ParamRaw: json.RawMessage(`{"id":1}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetMode() },
			want:    (*prime.Mode)(nil),
			wantErr: true,
		},
		{
			name:    "notify with Timer",
			notify:  &prime.Notify{Component: prime.ComponentTimer, ParamRaw: json.RawMessage(`{"id":1}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetTimer() },
			want:    &prime.Timer{ID: 1},
			wantErr: false,
		},
		{
			name:    "notify with without Timer",
			notify:  &prime.Notify{Component: prime.ComponentState},
			call:    func(n *prime.Notify) (any, error) { return n.GetTimer() },
			want:    (*prime.Timer)(nil),
			wantErr: false,
		},
		{
			name:    "notify with corrupted Timer",
			notify:  &prime.Notify{Component: prime.ComponentTimer, ParamRaw: json.RawMessage(`{"id":"1"}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetTimer() },
			want:    (*prime.Timer)(nil),
			wantErr: true,
		},
		{
			name:    "notify with services",
			notify:  &prime.Notify{Component: prime.ComponentService, ParamRaw: json.RawMessage(`{}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetService() },
			want:    &prime.VinculumServices{},
			wantErr: false,
		},
		{
			name:    "notify with without services",
			notify:  &prime.Notify{Component: prime.ComponentState},
			call:    func(n *prime.Notify) (any, error) { return n.GetService() },
			want:    (*prime.VinculumServices)(nil),
			wantErr: false,
		},
		{
			name:    "notify with corrupted services",
			notify:  &prime.Notify{Component: prime.ComponentService, ParamRaw: json.RawMessage(`{"fireAlarm":"1"}`)},
			call:    func(n *prime.Notify) (any, error) { return n.GetService() },
			want:    (*prime.VinculumServices)(nil),
			wantErr: true,
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.call(tc.notify)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.want, got)
		})
	}
}
