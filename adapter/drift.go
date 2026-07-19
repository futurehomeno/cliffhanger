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
// thing is built first, so a factory error aborts before the destructive DestroyThingByID.
func (a *adapter) RebuildChangedThings(seeds ThingSeeds) error {
	for _, seed := range seeds {
		live := a.ThingByID(seed.ID)
		if live == nil {
			continue
		}

		// The prospective thing reuses the live address so only genuine capability changes,
		// not address-derived fields, differ.
		prospective, err := a.factory.Create(a, a.publisher, newSeedState(seed, live.Address()))
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

		if err := a.DestroyThingByID(seed.ID); err != nil {
			return fmt.Errorf("rebuild %s: destroy: %w", seed.ID, err)
		}

		if err := a.CreateThing(seed); err != nil {
			return fmt.Errorf("rebuild %s: recreate (device excluded until next restart): %w", seed.ID, err)
		}
	}

	return nil
}

// topologyChecksum hashes only the service topology (address, groups, service specs), not
// cosmetic metadata like a product name, so a rebuild is triggered by a real capability
// change rather than by a rename on upgrade.
func topologyChecksum(report *fimptype.ThingInclusionReport) (uint32, error) {
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
