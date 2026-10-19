package adapter

import "slices"

// SeedsFromSelection builds seeds for the available devices matching the selected IDs, ready to
// be passed to EnsureThings. A nil selection selects every available device; a non-nil empty
// selection selects none - unmarshalling a missing JSON key yields nil, so callers that mean
// "none" must pass a non-nil empty slice. Duplicate entries yield a single seed: two seeds with
// the same ID would create two things, the second stranded at a second address.
func SeedsFromSelection[T any](available []T, selected []string, seed func(T) *ThingSeed) ThingSeeds {
	var seeds ThingSeeds

	for _, item := range available {
		s := seed(item)
		if s == nil {
			continue
		}

		if selected != nil && !slices.Contains(selected, s.ID) {
			continue
		}

		if seeds.Contains(s.ID) {
			continue
		}

		seeds = append(seeds, s)
	}

	return seeds
}
