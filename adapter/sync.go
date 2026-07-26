package adapter

import (
	"errors"
	"fmt"
)

// SyncThings reconciles things to match the devices selected from a freshly fetched list and
// returns the seeds it applied, ready to be passed to RebuildChangedThings without fetching
// again.
//
// The fetch runs outside the adapter lock and is the sole authority. If it fails nothing is
// mutated and the error is returned, so a network glitch can never wipe live things. If it
// succeeds the response is complete by definition: every selected device absent from it is
// destroyed, including when the response is empty. Adapters must therefore make their client
// return an error - never a truncated or empty slice - on a non-2xx, a rate limit or an
// unparsable body.
//
// A nil selection selects every available device; a non-nil empty selection selects none.
//
// Selected devices that vanished from the response and that the adapter does not own still get
// an exclusion report, so an upgraded hub can drop stale nodes a legacy adapter left behind.
// The device ID is used as the address of that report, which is only meaningful for adapters
// pinning ThingSeed.CustomAddress to the device ID; it is suppressed whenever the ID is already
// an ID or an address the adapter owns, so it can never exclude an unrelated thing.
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
) (ThingSeeds, error) {
	available, err := fetch()
	if err != nil {
		return nil, fmt.Errorf("adapter: fetch devices: %w", err)
	}

	seeds := SeedsFromSelection(available, selected, seed)

	var errs []error

	errs = append(errs, excludeVanishedDevices(a, selected, seeds)...)

	if err := a.EnsureThings(seeds); err != nil {
		errs = append(errs, err)
	}

	return seeds, errors.Join(errs...)
}

// excludeVanishedDevices announces an exclusion for every selected device the fetch no longer
// lists and the adapter owns no thing for. It must run before EnsureThings: afterwards the
// store entry is gone, so every legitimately destroyed device would look orphaned and be
// excluded a second time at the wrong address.
func excludeVanishedDevices(a Adapter, selected []string, seeds ThingSeeds) []error {
	var errs []error

	for _, id := range selected {
		if seeds.Contains(id) {
			continue
		}

		if _, owned := a.ExchangeID(id); owned {
			continue // EnsureThings destroys it and announces its real address.
		}

		if _, taken := a.ExchangeAddress(id); taken {
			continue // The ID is another device's address; excluding it would kill that thing.
		}

		if err := a.DestroyThingByAddress(id); err != nil {
			errs = append(errs, fmt.Errorf("adapter: exclude vanished device %s: %w", id, err))
		}
	}

	return errs
}
