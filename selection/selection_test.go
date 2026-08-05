package selection_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/selection"
)

func TestSelection_NilMeansAllEmptyMeansNone(t *testing.T) {
	t.Parallel()

	var all selection.Selection

	assert.True(t, all.IncludeAll())
	assert.True(t, all.Contains("anything"))

	none := selection.Selection{}

	assert.False(t, none.IncludeAll())
	assert.False(t, none.Contains("anything"))

	some := selection.Selection{"a"}

	assert.False(t, some.IncludeAll())
	assert.True(t, some.Contains("a"))
	assert.False(t, some.Contains("b"))
}

func TestSelection_CopiesPreserveTheDistinction(t *testing.T) {
	t.Parallel()

	// The trap this type exists to avoid: the idiomatic copy collapses empty to nil, which
	// would silently turn "the user deselected everything" into "include every device".
	require.Nil(t, append([]string(nil), selection.Selection{}...))

	assert.NotNil(t, selection.Selection{}.Clone())
	assert.False(t, selection.Selection{}.Clone().IncludeAll())
	assert.Nil(t, selection.Selection(nil).Clone())

	empty, removed := selection.Selection{}.Without("a")

	assert.False(t, removed)
	assert.False(t, empty.IncludeAll())

	// Removing the last device leaves "none", never "all".
	last, removed := selection.Selection{"a"}.Without("a")

	assert.True(t, removed)
	assert.False(t, last.IncludeAll())
	assert.Empty(t, last)

	// "Every device" cannot express an exclusion.
	unchanged, removed := selection.Selection(nil).Without("a")

	assert.False(t, removed)
	assert.True(t, unchanged.IncludeAll())

	remaining, removed := selection.Selection{"a", "b"}.Without("a")

	assert.True(t, removed)
	assert.Equal(t, selection.Selection{"b"}, remaining)
}

func TestSelection_SurvivesJSONInBothDirections(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		raw        string
		includeAll bool
		want       string
	}{
		{"absent key", `{}`, true, `{"selected_devices":null}`},
		{"null", `{"selected_devices":null}`, true, `{"selected_devices":null}`},
		{"empty list", `{"selected_devices":[]}`, false, `{"selected_devices":[]}`},
		{"populated list", `{"selected_devices":["a"]}`, false, `{"selected_devices":["a"]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var devices selection.Devices

			require.NoError(t, json.Unmarshal([]byte(tc.raw), &devices))
			assert.Equal(t, tc.includeAll, devices.Selection().IncludeAll())

			out, err := json.Marshal(devices)

			require.NoError(t, err)
			assert.JSONEq(t, tc.want, string(out))
		})
	}
}
