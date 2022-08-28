package config

import (
	"fmt"
	"strings"
)

// ValidateSetting is a helper that perform validation of a setting before passing it to a dedicated setter.
func ValidateSetting[T any](setter func(T) error, validators ...func(T) error) func(T) error {
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

// SettingWithin is a setting validator comparing a setting value against a set of allowed values.
func SettingWithin[T comparable](list []T) func(T) error {
	return func(val T) error {
		for _, v := range list {
			if v == val {
				return nil
			}
		}

		allowed := make([]string, len(list))
		for i, v := range list {
			allowed[i] = fmt.Sprintf("%+v", v)
		}

		return fmt.Errorf("config: value is not within the list of allowed values: %s", strings.Join(allowed, ", "))
	}
}
