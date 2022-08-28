package config_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/config"
)

func TestValidateSetting(t *testing.T) {
	t.Parallel()

	success := func(s string) error {
		if s != "valid_value" {
			return errors.New("test")
		}

		return nil
	}

	failure := func(s string) error {
		if s == "valid_value" {
			return errors.New("test")
		}

		return nil
	}

	tt := []struct {
		name      string
		setter    func(string) error
		validator func(string) error
		argument  string
		wantErr   bool
	}{
		{
			name:      "valid value",
			setter:    success,
			validator: success,
			argument:  "valid_value",
			wantErr:   false,
		},
		{
			name:      "valid value with persistence error",
			setter:    failure,
			validator: success,
			argument:  "valid_value",
			wantErr:   true,
		},
		{
			name:      "invalid value",
			setter:    success,
			validator: failure,
			argument:  "invalid_value",
			wantErr:   true,
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			validate := config.ValidateSetting(tc.setter, tc.validator)

			err := validate(tc.argument)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSettingWithin(t *testing.T) {
	t.Parallel()

	validator := config.SettingWithin([]string{"a", "b", "c"})

	assert.NoError(t, validator("a"))

	assert.Error(t, validator("d"))
}
