package adapter

import (
	"errors"
	"fmt"
	"slices"
)

var ErrUnmappedDevice = errors.New("adapter: seed function could not map a device")

// SyncThings reconciles things to match the devices selected from a freshly fetched list and
// returns the seeds it applied, ready to be passed to RebuildChangedThings without fetching
// again.
//
// The fetch runs outside the adapter lock and is the sole authority. If it fails nothing is
// mutated and the error is returned, so a network glitch can never wipe live things. If it
// succeeds the response is complete by definition: every selected device absent from it is
// destroyed, including when the response is empty, unless an item maps to a nil seed. Adapters must therefore make their client
// return an error - never a truncated or empty slice - on a non-2xx, a rate limit or an
// unparsable body.
//
// A nil selection selects every available device; a non-nil empty selection selects none.
//
// A seed function returning nil means the item cannot be mapped now. Its ID is unknown, so no
// unseeded selected device can be told apart from it: while any item maps to nil, no vanished
// device is excluded, the live things of unseeded selected devices are kept, and the returned
// error wraps ErrUnmappedDevice, so callers do not prune their selection and the stale sweep is
// skipped. Devices the adapter never supports must be filtered out in fetch, not by a nil seed,
// or no sync would clean up again.
//
// Selected devices that vanished from the response and that the adapter does not own still get
// an exclusion report, so an upgraded hub can drop stale nodes a legacy adapter left behind.
// The device ID is used as the address of that report, which is only meaningful for adapters
// pinning ThingSeed.CustomAddress to the device ID; it is suppressed whenever the ID is already
// an ID or an address the adapter owns, so it can never exclude an unrelated thing. The excluded
// IDs are returned so the caller can drop them from its persisted selection - otherwise the same
// vanished device is re-announced on every subsequent sync, since nothing here mutates selected.
//
// Every pass is best-effort per device: failures are collected with errors.Join and the seeds
// are still returned, so a single broken device blocks neither the rest of the sync nor a
// follow-up drift rebuild.
//
// SyncThings is not atomic: it takes and releases the adapter lock per operation. Adapters that
// also handle cmd.thing.delete or configuration writes must serialise those against the sync
// with a shared router.MessageHandlerLocker.
func SyncThings[T any](
	a Adapter,
	fetch func() ([]T, error),
	selected []string,
	seed func(T) *ThingSeed,
) (seeds ThingSeeds, excludedIDs []string, err error) {
	available, err := fetch()
	if err != nil {
		return nil, nil, fmt.Errorf("adapter: fetch devices: %w", err)
	}

	unmapped := 0
	seeds = SeedsFromSelection(available, selected, func(item T) *ThingSeed {
		s := seed(item)
		if s == nil {
			unmapped++
		}

		return s
	})

	var errs []error

	ensure := seeds

	if unmapped > 0 {
		errs = append(errs, fmt.Errorf("%w: %d items", ErrUnmappedDevice, unmapped))
		ensure = append(slices.Clone(seeds), unseededLiveThings(a, selected, seeds)...)
	} else {
		var exclErrs []error

		excludedIDs, exclErrs = excludeVanishedDevices(a, selected, seeds)
		errs = append(errs, exclErrs...)
	}

	if err := a.EnsureThings(ensure); err != nil {
		errs = append(errs, err)
	}

	// Only once the whole reconciliation succeeded are the things known to match the fetched
	// list, which is what makes a hub node the adapter owns nothing for provably stale.
	if impl, ok := a.(*adapter); ok && len(errs) == 0 {
		impl.excludeStaleNodesOnce()
	}

	return seeds, excludedIDs, errors.Join(errs...)
}

// unseededLiveThings returns a seed pinning every live thing of a selected device that has no
// seed. EnsureThings never destroys, creates or heals from such a seed.
func unseededLiveThings(a Adapter, selected []string, seeds ThingSeeds) ThingSeeds {
	var kept ThingSeeds

	for _, t := range a.Things() {
		id, ok := a.ExchangeAddress(t.Address())
		if ok && !seeds.Contains(id) && (selected == nil || slices.Contains(selected, id)) {
			kept = append(kept, &ThingSeed{ID: id, CustomAddress: t.Address(), keep: true})
		}
	}

	return kept
}

// excludeVanishedDevices announces an exclusion for every selected device the fetch no longer
// lists and the adapter owns no thing for, and returns the IDs it successfully excluded. It must
// run before EnsureThings: afterwards the store entry is gone, so every legitimately destroyed
// device would look orphaned and be excluded a second time at the wrong address.
func excludeVanishedDevices(a Adapter, selected []string, seeds ThingSeeds) ([]string, []error) {
	var excluded []string

	var errs []error

	// Nothing dedupes the selection on the way into the configuration, and an ID repeated there
	// would otherwise announce the same vanished device twice.
	seen := make(map[string]struct{}, len(selected))

	for _, id := range selected {
		if seeds.Contains(id) {
			continue
		}

		if _, done := seen[id]; done {
			continue
		}

		seen[id] = struct{}{}

		if _, owned := a.ExchangeID(id); owned {
			continue // EnsureThings destroys it and announces its real address.
		}

		if _, taken := a.ExchangeAddress(id); taken {
			continue // The ID is another device's address; excluding it would kill that thing.
		}

		if err := a.DestroyThingByAddress(id); err != nil {
			errs = append(errs, fmt.Errorf("adapter: exclude vanished device %s: %w", id, err))

			continue
		}

		excluded = append(excluded, id)
	}

	return excluded, errs
}
