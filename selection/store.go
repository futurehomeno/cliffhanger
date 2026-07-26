package selection

// Store reads and writes a Selection inside an application configuration. It borrows the
// configuration service's lock and its configured-at stamping rather than introducing a second
// lock over the same model.
type Store struct {
	get func() Selection
	set func(Selection) error
}

// NewStore wires a Store to a configuration service's read and write paths. get must read a copy
// under the service's read lock; set must apply the value under its write lock, stamp the
// configuration time and save.
func NewStore(get func() Selection, set func(Selection) error) *Store {
	return &Store{get: get, set: set}
}

// Get returns the selection, preserving the nil/empty distinction.
func (s *Store) Get() Selection { return s.get() }

// Set replaces the selection with a copy of the provided one. The copy defends against a caller
// mutating the slice it handed over.
func (s *Store) Set(sel Selection) error { return s.set(sel.Clone()) }

// Remove drops a device from the selection, e.g. after cmd.thing.delete, so the device is not
// recreated by the next sync. It is a no-op - no write, no stamp - when the device is not
// selected or when the selection includes every device.
//
// The read-then-write is not atomic, so it must run under the same handler lock as the
// configuration writes it can race, which adapter.WithLocker provides.
func (s *Store) Remove(id string) error {
	next, removed := s.get().Without(id)
	if !removed {
		return nil
	}

	return s.set(next)
}
