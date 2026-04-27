package config

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// Migration describes a single version upgrade step for a config_version
// string. Migrations are applied in a chain: as long as some step's From
// matches the current version, it runs and the version advances to its To.
type Migration struct {
	// From is the version expected to trigger this step. Use "" to migrate
	// an unversioned or fresh configuration.
	From int
	// To is the version written after Do succeeds.
	To int
	// Do performs the migration work. Typically a closure that mutates the
	// consumer's config. Optional: leave nil for a pure version bump.
	Do func() error
}

// Migrate advances ConfigVersion through a chain of migrations. It
// repeatedly picks a step whose From matches the current ConfigVersion,
// executes Do, and writes To into ConfigVersion. Returns the number of
// applied steps (0 when already up-to-date).
//
// Migrate only mutates the in-memory config; persisting the result is the
// caller's responsibility (typically via storage.Save after the call).
func (d *Default) Migrate(migrations ...Migration) (int, error) {
	applied := 0
	seen := make(map[int]bool)

	for {
		current := d.ConfigVersion

		idx := -1

		for i, m := range migrations {
			if m.From == current {
				idx = i

				break
			}
		}

		if idx == -1 {
			return applied, nil
		}

		m := migrations[idx]
		if m.To == m.From {
			return applied, fmt.Errorf("config: migration %d->%d does not advance version", m.From, m.To)
		}

		if seen[m.To] {
			return applied, fmt.Errorf("config: migration cycle detected at version %d", m.To)
		}

		seen[current] = true

		log.Infof("[cliff] migrating config from %d to %d", m.From, m.To)

		if m.Do != nil {
			if err := m.Do(); err != nil {
				return applied, fmt.Errorf("config: migration %d->%d failed: %w", m.From, m.To, err)
			}
		}

		d.ConfigVersion = m.To
		applied++
	}
}
