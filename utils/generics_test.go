package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/utils"
)

type unit string

func TestNormalize(t *testing.T) {
	t.Parallel()

	allowed := []unit{"W", "kWh"}

	got, ok := utils.Normalize(unit("kwh"), allowed)
	assert.True(t, ok)
	assert.Equal(t, unit("kWh"), got)

	got, ok = utils.Normalize(unit("A"), allowed)
	assert.False(t, ok)
	assert.Empty(t, got)

	got, ok = utils.Normalize(unit("W"), nil)
	assert.False(t, ok)
	assert.Empty(t, got)
}
