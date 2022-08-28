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

		for _, v := range values {
			if v == val {
				return nil
			}
		}

		allowed := make([]string, len(values))
		for i, v := range values {
			allowed[i] = fmt.Sprintf("%+v", v)
		}

		return fmt.Errorf("config: value is not within the list of allowed values: %s", strings.Join(allowed, ", "))
	}
}

// Between is a setting validator comparing a setting value against a minimum and maximum allowed values.
func Between[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64](min, max T) func(T) error {
	return func(val T) error {
		if val < min {
			return fmt.Errorf("config: provided value %v is lesser than the minimum allowed value: %v", val, min)
		}

		if val > max {
			return fmt.Errorf("config: provided value %v is greater than the maximum allowed value: %v", val, max)
		}

		return nil
	}
}
