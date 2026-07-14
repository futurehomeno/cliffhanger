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
