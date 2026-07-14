package adapter

import "slices"

// SeedsFromSelection builds seeds for the available devices matching the selected IDs,
// ready to be passed to EnsureThings.
func SeedsFromSelection[T any](available []T, selected []string, seed func(T) *ThingSeed) ThingSeeds {
	var seeds ThingSeeds

	for _, item := range available {
		s := seed(item)
		if slices.Contains(selected, s.ID) {
			seeds = append(seeds, s)
		}
	}

	return seeds
}
