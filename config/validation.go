package config

import (
	"fmt"
	"reflect"
	"strings"
)

// Validate is a helper that perform validation of a setting before passing it to a dedicated setter.
func Validate[T any](setter func(T) error, validators ...func(T) error) func(T) error {
	return func(val T) error {
		for _, v := range validators {
			err := v(val)
			if err != nil {
				return fmt.Errorf("config: failed to validate setting: %w", err)
			}
		}

		return setter(val)
	}
}

// Within is a setting validator comparing a setting value against a set of allowed values.
func Within[T comparable](values []T, optional bool) func(T) error {
	return func(val T) error {
		if optional && reflect.ValueOf(val).IsZero() {
			return nil
		}

		if Contains(val, values) {
			return nil
		}

		allowed := make([]string, len(values))
		for i, v := range values {
			allowed[i] = fmt.Sprintf("%+v", v)
		}

		return fmt.Errorf("config: value is not within the list of allowed values: %s", strings.Join(allowed, ", "))
	}
}

// Between is a setting validator comparing a setting value against a minimum and maximum allowed values.
func Between[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64](minimum, maximum T) func(T) error {
	return func(val T) error {
		if val < minimum {
			return fmt.Errorf("config: provided value %v is lesser than the minimum allowed value: %v", val, minimum)
		}

		if val > maximum {
			return fmt.Errorf("config: provided value %v is greater than the maximum allowed value: %v", val, maximum)
		}

		return nil
	}
}

// Lesser is a setting validator comparing a setting value against a maximum allowed value.
func Lesser[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64](than T) func(T) error {
	return func(val T) error {
		if val >= than {
			return fmt.Errorf("config: provided value %v must be lesser than than the maximum allowed value: %v", val, than)
		}

		return nil
	}
}

// Greater is a setting validator comparing a setting value against a minimum allowed value.
func Greater[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64](than T) func(T) error {
	return func(val T) error {
		if val <= than {
			return fmt.Errorf("config: provided value %v must be greater than the minimum allowed value: %v", val, than)
		}

		return nil
	}
}

// Contains is a helper that checks if a value is present in a slice.
func Contains[T comparable](needle T, haystack []T) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}

	return false
}

// Deduplicate is a helper that deduplicates a slice.
func Deduplicate[T comparable](list []T) []T {
	deduplicated := make(map[T]struct{})

	for _, el := range list {
		deduplicated[el] = struct{}{}
	}

	result := make([]T, len(deduplicated))

	i := 0

	for el := range deduplicated {
		result[i] = el

		i++
	}

	return result
}
