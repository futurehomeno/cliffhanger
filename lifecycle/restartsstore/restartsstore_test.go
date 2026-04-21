package restartsstore_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/database"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/lifecycle/restartsstore"
)

func newTestDatabase(t *testing.T) database.Database {
	t.Helper()

	db, err := database.NewDatabase(t.TempDir())
	require.NoError(t, err)

	require.NoError(t, db.Start())
	t.Cleanup(func() { _ = db.Stop() })

	return db
}

func TestStore_ImplementsRestartsStore(t *testing.T) {
	t.Parallel()

	var _ lifecycle.RestartsStore = (*restartsstore.Store)(nil)
}

func TestStore_GetReturnsZeroWhenEmpty(t *testing.T) {
	t.Parallel()

	db := newTestDatabase(t)
	store := restartsstore.New(db)

	n, err := store.GetRestartsCount()

	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestStore_SetThenGetRoundTrips(t *testing.T) {
	t.Parallel()

	db := newTestDatabase(t)
	store := restartsstore.New(db)

	require.NoError(t, store.SetRestartsCount(7))

	n, err := store.GetRestartsCount()

	require.NoError(t, err)
	assert.Equal(t, 7, n)
}

func TestStore_SurvivesReopen(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()

	db1, err := database.NewDatabase(workDir)
	require.NoError(t, err)
	require.NoError(t, db1.Start())

	require.NoError(t, restartsstore.New(db1).SetRestartsCount(42))
	require.NoError(t, db1.Stop())

	// Reopen the same directory: the persisted value must come back.
	db2, err := database.NewDatabase(workDir)
	require.NoError(t, err)
	require.NoError(t, db2.Start())
	t.Cleanup(func() { _ = db2.Stop() })

	n, err := restartsstore.New(db2).GetRestartsCount()

	require.NoError(t, err)
	assert.Equal(t, 42, n)
}

func TestStore_LoadRestartsCount_IntegratesWithLifecycle(t *testing.T) {
	t.Parallel()

	db := newTestDatabase(t)
	store := restartsstore.New(db)

	// First boot: count goes 0 -> 1.
	l1 := lifecycle.New(nil)
	require.NoError(t, l1.LoadRestartsCount(store))
	assert.Equal(t, 1, l1.RestartsCount())

	// Simulated second boot against the same store: 1 -> 2.
	l2 := lifecycle.New(nil)
	require.NoError(t, l2.LoadRestartsCount(store))
	assert.Equal(t, 2, l2.RestartsCount())
}
