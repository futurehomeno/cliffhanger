package restartsstore

import (
	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/database"
)

const (
	dbBucket = "restarts"
	dbKey    = "count"
)

// RestartsStore persists and increments the restart counter across application
// restarts. A missing value must be treated as zero.
type RestartsStore interface {
	IncrementRestartsCount() (int, error)
}

// Store is a database-backed RestartsStore.
type Store struct {
	db database.Database
}

// New returns a database-backed Store.
func New(db database.Database) *Store {
	return &Store{db: db}
}

// IncrementRestartsCount implements RestartsStore.
func (s *Store) IncrementRestartsCount() (int, error) {
	n, err := s.GetRestartsCount()
	if err != nil {
		return 0, err
	}

	n++

	return n, s.SetRestartsCount(n)
}

// GetRestartsCount returns the current persisted restart count.
func (s *Store) GetRestartsCount() (int, error) {
	var n int

	ok, err := s.db.Get(dbBucket, dbKey, &n)
	if err != nil || !ok {
		return 0, err
	}

	return n, nil
}

// SetRestartsCount overwrites the persisted restart count.
func (s *Store) SetRestartsCount(n int) error {
	return s.db.Set(dbBucket, dbKey, n)
}

// NewDefault returns a config-backed RestartsStore. The accessor must return
// a pointer to the embedded Default block; save persists any mutation to disk.
func NewDefault(accessor func() *config.Default, save func() error) RestartsStore {
	return &defaultStore{accessor: accessor, save: save}
}

type defaultStore struct {
	accessor func() *config.Default
	save     func() error
}

func (s *defaultStore) IncrementRestartsCount() (int, error) {
	n := s.accessor().RestartsCount + 1
	s.accessor().RestartsCount = n

	return n, s.save()
}
