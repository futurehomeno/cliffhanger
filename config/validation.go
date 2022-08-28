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
