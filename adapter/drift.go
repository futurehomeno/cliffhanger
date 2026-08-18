package adapter

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"

	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"
)

// RebuildChangedThings rebuilds any already-registered thing whose seed would produce a
// different service topology than the live thing. EnsureThings only reconciles presence, so
// a device that gains a capability keeps its old services until it is rebuilt.
//
// Best-effort per seed: failures are joined. The whole pass runs under the adapter lock, so
// nothing can destroy a thing between the lookup and its rebuild.
func (a *adapter) RebuildChangedThings(seeds ThingSeeds) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	var errs []error

	for _, seed := range seeds {
		if err := a.rebuildChangedThing(seed); err != nil {
			errs = append(errs, fmt.Errorf("rebuild %s: %w", seed.ID, err))
		}
	}

	return errors.Join(errs...)
}

// rebuildChangedThing rebuilds a single registered thing if its service topology drifted.
// Assumes the adapter lock is held.
func (a *adapter) rebuildChangedThing(seed *ThingSeed) error {
	ts := a.state.byID(seed.ID)
	if ts == nil {
		return nil
	}

	live := a.things[ts.Address()]
	if live == nil {
		return nil
	}

	// Reuse the live address so only genuine capability changes, not address-derived fields,
	// differ. Building before the destroy also means a factory error costs nothing - but a
	// successful build is thrown away below if the topology is unchanged, so the factory must
	// hold to the no-persistent-side-effects precondition documented on ThingFactory.Create.
	prospective, err := a.factory.Create(a, a.publisher, newSeedState(seed, ts.Address()))
	if err != nil {
		return fmt.Errorf("build prospective thing: %w", err)
	}

	fresh, err := topologyChecksum(prospective.InclusionReport())
	if err != nil {
		return fmt.Errorf("checksum prospective: %w", err)
	}

	current, err := topologyChecksum(live.InclusionReport())
	if err != nil {
		return fmt.Errorf("checksum live: %w", err)
	}

	if fresh == current {
		return nil
	}

	// destroyThing drops the whole state record, so capture any persisted per-thing state and
	// restore it after the swap; otherwise a rebuild would silently discard it.
	var savedState json.RawMessage
	if err := ts.State(&savedState); err != nil {
		return fmt.Errorf("read state: %w", err)
	}

	// Preserve the live address so the rebuilt thing keeps its topic identity; a seed without
	// a CustomAddress would be assigned a fresh one on recreation.
	rebuildSeed := &ThingSeed{ID: seed.ID, CustomAddress: ts.Address(), Info: seed.Info}

	// destroyThing reporting an error has either completed the removal (only the exclusion
	// announcement failed) or kept the record after a failed state write; either way state.add
	// in the recreate below overwrites it. Aborting instead would leave the device gone or
	// ghosted until the next sync - and discard savedState with it.
	if err := a.destroyThing(ts.Address()); err != nil {
		log.Warnf("[adapter] Rebuild thing %s: destroy reported errors, recreating anyway. err: %v", seed.ID, err)
	}

	if err := a.createThing(rebuildSeed); err != nil {
		return fmt.Errorf("recreate (device excluded until next restart): %w", err)
	}

	if len(savedState) > 0 {
		if newTS := a.state.byID(seed.ID); newTS != nil {
			if err := newTS.SetState(savedState); err != nil {
				return fmt.Errorf("restore state: %w", err)
			}
		}
	}

	return nil
}

// topologyChecksum hashes the inclusion report's address, groups and full service specs so a
// rebuild fires when a device's capabilities change. It serializes the whole fimptype.Service,
// so a change to service metadata (alias, props, tags) also flips the checksum; that only ever
// causes an extra rebuild, never a missed one, and a rebuild preserves the thing's state.
func topologyChecksum(report *fimptype.ThingInclusionReport) (uint32, error) {
	if report == nil {
		return 0, nil
	}

	topology := struct {
		Address  string             `json:"address"`
		Groups   []string           `json:"groups"`
		Services []fimptype.Service `json:"services"`
	}{report.Address, report.Groups, report.Services}

	data, err := json.Marshal(topology)
	if err != nil {
		return 0, err
	}

	return crc32.ChecksumIEEE(data), nil
}

// seedState is an in-memory ThingState carrying a seed's information, used only to build a
// prospective inclusion report without registering or persisting anything.
type seedState struct {
	id      string
	address string
	info    any
}

func newSeedState(seed *ThingSeed, address string) *seedState {
	return &seedState{id: seed.ID, address: address, info: seed.Info}
}

func (s *seedState) ID() string      { return s.id }
func (s *seedState) Address() string { return s.address }

func (s *seedState) Info(model any) error {
	if s.info == nil {
		return nil
	}

	data, err := json.Marshal(s.info)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, model)
}

func (s *seedState) State(any) error                   { return nil }
func (s *seedState) SetState(any) error                { return nil }
func (s *seedState) InclusionChecksum() uint32         { return 0 }
func (s *seedState) SetInclusionChecksum(uint32) error { return nil }
