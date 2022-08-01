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
