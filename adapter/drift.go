package adapter

import (
	"encoding/json"
	"fmt"
	"hash/crc32"

	"github.com/futurehomeno/fimpgo/fimptype"
)

// RebuildChangedThings rebuilds any already-registered thing whose seed would produce a
// different service topology than the live thing. EnsureThings only reconciles presence, so
// a device that gains a capability keeps its old services until it is rebuilt. The prospective
// thing is built first, so a factory error aborts before the destructive destroy. The whole
// pass runs under the adapter lock, matching EnsureThings, so no concurrent operation can
// destroy a thing between the lookup and its rebuild.
func (a *adapter) RebuildChangedThings(seeds ThingSeeds) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	for _, seed := range seeds {
		ts := a.state.byID(seed.ID)
		if ts == nil {
			continue
		}

		live := a.things[ts.Address()]
		if live == nil {
			continue
		}

		// The prospective thing reuses the live address so only genuine capability changes,
		// not address-derived fields, differ.
		prospective, err := a.factory.Create(a, a.publisher, newSeedState(seed, ts.Address()))
		if err != nil {
			return fmt.Errorf("rebuild %s: build prospective thing: %w", seed.ID, err)
		}

		fresh, err := topologyChecksum(prospective.InclusionReport())
		if err != nil {
			return fmt.Errorf("rebuild %s: checksum prospective: %w", seed.ID, err)
		}

		current, err := topologyChecksum(live.InclusionReport())
		if err != nil {
			return fmt.Errorf("rebuild %s: checksum live: %w", seed.ID, err)
		}

		if fresh == current {
			continue
		}

		// destroyThing drops the whole state record and createThing rebuilds only
		// ID/Address/Info, so capture any persisted per-thing state and restore it after the
		// swap; otherwise a topology rebuild would silently discard it.
		var savedState json.RawMessage
		if err := ts.State(&savedState); err != nil {
			return fmt.Errorf("rebuild %s: read state: %w", seed.ID, err)
		}

		// Preserve the live address so the rebuilt thing keeps its topic identity; a seed
		// without a CustomAddress would otherwise be assigned a fresh address on recreation.
		rebuildSeed := &ThingSeed{ID: seed.ID, CustomAddress: ts.Address(), Info: seed.Info}

		if err := a.destroyThing(ts.Address()); err != nil {
			return fmt.Errorf("rebuild %s: destroy: %w", seed.ID, err)
		}

		if err := a.createThing(rebuildSeed); err != nil {
			return fmt.Errorf("rebuild %s: recreate (device excluded until next restart): %w", seed.ID, err)
		}

		if len(savedState) > 0 {
			if newTS := a.state.byID(seed.ID); newTS != nil {
				if err := newTS.SetState(savedState); err != nil {
					return fmt.Errorf("rebuild %s: restore state: %w", seed.ID, err)
				}
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
