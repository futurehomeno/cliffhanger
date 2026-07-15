package adapter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter"
)

func TestSeedsFromSelection(t *testing.T) {
	t.Parallel()

	type device struct{ ID, Name string }

	available := []device{{"1", "a"}, {"2", "b"}, {"3", "c"}}

	seeds := adapter.SeedsFromSelection(available, []string{"1", "3", "missing"}, func(d device) *adapter.ThingSeed {
		return &adapter.ThingSeed{ID: d.ID, Info: d.Name}
	})

	assert.Len(t, seeds, 2)
	assert.True(t, seeds.Contains("1"))
	assert.True(t, seeds.Contains("3"))
	assert.False(t, seeds.Contains("2"))

	assert.Empty(t, adapter.SeedsFromSelection(available, nil, func(d device) *adapter.ThingSeed {
		return &adapter.ThingSeed{ID: d.ID}
	}))
}
