package backoff_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/backoff"
)

func TestNewTolerantFixed(t *testing.T) {
	t.Parallel()

	b := backoff.NewTolerantFixed(2, 5*time.Minute)

	assert.Equal(t, time.Duration(0), b.Next(), "first failure should not be delayed")
	assert.Equal(t, time.Duration(0), b.Next(), "second failure should not be delayed")
	assert.Equal(t, 5*time.Minute, b.Next(), "third failure should be delayed")
	assert.Equal(t, 5*time.Minute, b.Next(), "further failures should keep the fixed delay")

	b.Reset()
	assert.Equal(t, time.Duration(0), b.Next(), "reset should restore tolerance")
}
