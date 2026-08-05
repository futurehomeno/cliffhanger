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

	// A nil selection is an application that has never been configured: include everything.
	assert.Len(t, adapter.SeedsFromSelection(available, nil, func(d device) *adapter.ThingSeed {
		return &adapter.ThingSeed{ID: d.ID}
	}), 3)

	// A non-nil empty selection is a user who deselected everything: include nothing.
	assert.Empty(t, adapter.SeedsFromSelection(available, []string{}, func(d device) *adapter.ThingSeed {
		return &adapter.ThingSeed{ID: d.ID}
	}))

	// A repeated third-party entry must not create a second thing at a second address.
	duplicated := []device{{"1", "a"}, {"1", "a"}}

	assert.Len(t, adapter.SeedsFromSelection(duplicated, []string{"1"}, func(d device) *adapter.ThingSeed {
		return &adapter.ThingSeed{ID: d.ID}
	}), 1)
}
