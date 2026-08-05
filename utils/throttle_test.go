package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/utils"
)

func TestThrottle(t *testing.T) {
	t.Parallel()

	var throttle utils.Throttle

	emitted := 0
	emit := func() { emitted++ }

	throttle.Do("a", emit)
	throttle.Do("a", emit)
	assert.Equal(t, 1, emitted, "repeated key should emit once")

	throttle.Do("b", emit)
	assert.Equal(t, 2, emitted, "new key should emit")

	throttle.Reset()
	throttle.Do("b", emit)
	assert.Equal(t, 3, emitted, "reset should allow the same key to emit again")
}

func TestThrottle_FirstEmptyKeyEmits(t *testing.T) {
	t.Parallel()

	var throttle utils.Throttle

	emitted := 0
	emit := func() { emitted++ }

	// The zero value of last is "" too, so a naive key==last check would mistake this first
	// call for a repeat of itself and silently skip it.
	throttle.Do("", emit)
	assert.Equal(t, 1, emitted, "the first call, even with an empty key, must emit")

	throttle.Do("", emit)
	assert.Equal(t, 1, emitted, "a repeated empty key must still throttle")

	throttle.Reset()
	throttle.Do("", emit)
	assert.Equal(t, 2, emitted, "reset must allow an empty key to emit again")
}
