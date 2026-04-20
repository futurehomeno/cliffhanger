// Package restartsstore provides a lifecycle.RestartsStore implementation
// backed by database.Database.
package restartsstore

import (
	"fmt"

	"github.com/futurehomeno/cliffhanger/database"
	"github.com/futurehomeno/cliffhanger/lifecycle"
)

const (
	bucket = "lifecycle"
	key    = "restarts_count"
)

// Store is a database-backed lifecycle.RestartsStore.
type Store struct {
	db database.Database
}

// New creates a new database-backed restarts store.
func New(db database.Database) *Store {
	return &Store{db: db}
}

// GetRestartsCount reads the persisted restart count. A missing value returns
// (0, nil) so the first startup begins counting from zero.
func (s *Store) GetRestartsCount() (int, error) {
	var n int

	ok, err := s.db.Get(bucket, key, &n)
	if err != nil {
		return 0, fmt.Errorf("restartsstore: failed to read restart count: %w", err)
	}

	if !ok {
		return 0, nil
	}

	return n, nil
}

// SetRestartsCount persists the restart count.
func (s *Store) SetRestartsCount(n int) error {
	if err := s.db.Set(bucket, key, n); err != nil {
		return fmt.Errorf("restartsstore: failed to persist restart count: %w", err)
	}

	return nil
}

var _ lifecycle.RestartsStore = (*Store)(nil)
