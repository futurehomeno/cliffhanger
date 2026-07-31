package selection_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/selection"
)

// testStore returns a store over an in-memory selection along with the number of writes it
// performed, so a no-op removal can be told apart from one that stamps the configuration.
func testStore(initial selection.Selection) (*selection.Store, *selection.Selection, *int) {
	sel, writes := initial, 0

	store := selection.NewStore(
		func() selection.Selection { return sel },
		func(next selection.Selection) error { sel = next; writes++; return nil },
	)

	return store, &sel, &writes
}

func TestStore_GetSetRemove(t *testing.T) {
	t.Parallel()

	store, sel, writes := testStore(selection.Selection{"a", "b"})

	assert.Equal(t, selection.Selection{"a", "b"}, store.Get())

	require.NoError(t, store.Remove("a"))
	assert.Equal(t, selection.Selection{"b"}, *sel)
	assert.Equal(t, 1, *writes)

	// Removing a device that is not selected must not stamp the configuration time.
	require.NoError(t, store.Remove("missing"))
	assert.Equal(t, 1, *writes)

	require.NoError(t, store.Set(selection.Selection{"c"}))
	assert.Equal(t, selection.Selection{"c"}, store.Get())
}

func TestStore_SetCopiesTheCallersSlice(t *testing.T) {
	t.Parallel()

	store, sel, _ := testStore(nil)

	provided := selection.Selection{"a"}

	require.NoError(t, store.Set(provided))

	provided[0] = "mutated"

	assert.Equal(t, selection.Selection{"a"}, *sel)
}

func TestStore_RemoveOnIncludeAllIsNoOp(t *testing.T) {
	t.Parallel()

	// "Every device" cannot express an exclusion, so the selection is left alone rather than
	// being silently materialised into an explicit list.
	store, sel, writes := testStore(nil)

	require.NoError(t, store.Remove("a"))
	assert.True(t, sel.IncludeAll())
	assert.Equal(t, 0, *writes)
}

func TestStore_RemovePropagatesWriteFailure(t *testing.T) {
	t.Parallel()

	errWrite := errors.New("write failed")

	store := selection.NewStore(
		func() selection.Selection { return selection.Selection{"a"} },
		func(selection.Selection) error { return errWrite },
	)

	assert.ErrorIs(t, store.Remove("a"), errWrite)
}
