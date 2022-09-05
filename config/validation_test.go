package config_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/config"
)

func TestValidate(t *testing.T) {
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

			validate := config.Validate(tc.setter, tc.validator)

			err := validate(tc.argument)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWithin(t *testing.T) {
	t.Parallel()

	validator := config.Within([]string{"a", "b", "c"}, false)

	assert.NoError(t, validator("a"))
	assert.Error(t, validator("d"))
	assert.Error(t, validator(""))

	optionalValidator := config.Within([]string{"a", "b", "c"}, true)
	assert.NoError(t, optionalValidator("a"))
	assert.Error(t, optionalValidator("d"))
	assert.NoError(t, optionalValidator(""))
}

func TestBetween(t *testing.T) {
	t.Parallel()

	validator := config.Between(float64(5), float64(10))

	assert.NoError(t, validator(5))
	assert.NoError(t, validator(7))
	assert.NoError(t, validator(10))
	assert.Error(t, validator(0))
	assert.Error(t, validator(15))
}

func TestGreater(t *testing.T) {
	t.Parallel()

	validator := config.Greater(float64(5))

	assert.NoError(t, validator(6))
	assert.Error(t, validator(5))
	assert.Error(t, validator(4))
}

func TestLesser(t *testing.T) {
	t.Parallel()

	validator := config.Lesser(float64(5))

	assert.NoError(t, validator(4))
	assert.Error(t, validator(5))
	assert.Error(t, validator(6))
}
