package utils_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/utils"
)

func TestTimestamped(t *testing.T) {
	t.Parallel()

	var c utils.Timestamped[int]

	_, _, ok := c.Get()
	assert.False(t, ok, "empty holder should report no value")

	now := time.Now()

	assert.True(t, c.Set(1, now))
	assert.False(t, c.Set(2, now.Add(-time.Second)), "older value should be rejected")
	assert.True(t, c.Set(3, now), "equal timestamp should win (last-write-wins)")
	assert.True(t, c.Set(4, now.Add(time.Second)))

	value, ts, ok := c.Get()
	assert.True(t, ok)
	assert.Equal(t, 4, value)
	assert.Equal(t, now.Add(time.Second), ts)
}
