package utils

import "strings"

func Ptr[T any](val T) *T {
	return &val
}

// Normalize returns the canonical spelling of value among the allowed ones, matched
// case-insensitively.
func Normalize[T ~string](value T, allowed []T) (T, bool) {
	for _, a := range allowed {
		if strings.EqualFold(string(value), string(a)) {
			return a, true
		}
	}

	var zero T

	return zero, false
}
