package restartsstore_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/config"
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

	var _ restartsstore.RestartsStore = (*restartsstore.Store)(nil)
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

	db2, err := database.NewDatabase(workDir)
	require.NoError(t, err)
	require.NoError(t, db2.Start())
	t.Cleanup(func() { _ = db2.Stop() })

	n, err := restartsstore.New(db2).GetRestartsCount()

	require.NoError(t, err)
	assert.Equal(t, 42, n)
}

func TestStore_IncrementRestartsCount_IntegratesWithLifecycle(t *testing.T) {
	t.Parallel()

	db := newTestDatabase(t)
	store := restartsstore.New(db)

	// First boot: count goes 0 -> 1.
	l1 := lifecycle.New(store)
	assert.Equal(t, 1, l1.RestartsCount())

	// Simulated second boot against the same store: 1 -> 2.
	l2 := lifecycle.New(store)
	assert.Equal(t, 2, l2.RestartsCount())
}

func TestNewDefault_RoundTrip(t *testing.T) {
	t.Parallel()

	d := &config.Default{}
	saves := 0
	store := restartsstore.NewDefault(func() *config.Default { return d }, func() error { saves++; return nil })

	// First increment: 0 -> 1.
	n, err := store.IncrementRestartsCount()
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, 1, d.RestartsCount)
	assert.Equal(t, 1, saves)

	// Second increment: 1 -> 2.
	n, err = store.IncrementRestartsCount()
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, 2, d.RestartsCount)
}

func TestNewDefault_PropagatesSaveError(t *testing.T) {
	t.Parallel()

	d := &config.Default{}
	saveErr := errors.New("disk boom")
	store := restartsstore.NewDefault(func() *config.Default { return d }, func() error { return saveErr })

	_, err := store.IncrementRestartsCount()

	require.Error(t, err)
	assert.ErrorIs(t, err, saveErr)
}

func TestNewDefault_IntegratesWithLifecycle(t *testing.T) {
	t.Parallel()

	d := &config.Default{}
	store := restartsstore.NewDefault(func() *config.Default { return d }, func() error { return nil })

	l1 := lifecycle.New(store)
	assert.Equal(t, 1, l1.RestartsCount())
	assert.Equal(t, 1, d.RestartsCount)

	l2 := lifecycle.New(store)
	assert.Equal(t, 2, l2.RestartsCount())
	assert.Equal(t, 2, d.RestartsCount)
}
