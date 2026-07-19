package adapter

import "errors"

// ErrIncompleteFetch is returned by SyncThings when a fetched device list is missing a
// currently-registered selected device, which usually means a partial or empty third-party
// response. Reconciling on it would destroy live things, so it is skipped; callers typically
// log and retry rather than treat it as fatal.
var ErrIncompleteFetch = errors.New("adapter: incomplete device fetch, skipping reconcile")

// SyncThings reconciles things to match the selected devices from a freshly fetched list: it
// builds seeds with SeedsFromSelection and applies them via EnsureThings (create missing,
// destroy stale). As a guard against a partial or empty fetch wiping live things, it refuses
// to reconcile and returns ErrIncompleteFetch when a currently-registered selected device is
// absent from available. The fetch, selection and device-to-seed mapping stay caller-supplied.
func SyncThings[T any](a Adapter, available []T, selected []string, seed func(T) *ThingSeed) error {
	seeds := SeedsFromSelection(available, selected, seed)

	for _, id := range selected {
		if a.ThingByID(id) != nil && !seeds.Contains(id) {
			return ErrIncompleteFetch
		}
	}

	return a.EnsureThings(seeds)
}
