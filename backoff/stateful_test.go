package backoff_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/backoff"
)

func TestStateful_Next(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		backoff         backoff.Stateful
		expectedResults []time.Duration
	}{
		{
			name:            "regular backoff",
			backoff:         backoff.NewStateful(time.Second, 2*time.Second, 3*time.Second, 1, 2),
			expectedResults: []time.Duration{time.Second, 2 * time.Second, 2 * time.Second, 3 * time.Second, 3 * time.Second},
		},
		{
			name:            "no initial backoff",
			backoff:         backoff.NewStateful(time.Second, 2*time.Second, 3*time.Second, 0, 2),
			expectedResults: []time.Duration{2 * time.Second, 2 * time.Second, 3 * time.Second},
		},
		{
			name:            "no repeated backoff",
			backoff:         backoff.NewStateful(time.Second, 2*time.Second, 3*time.Second, 2, 0),
			expectedResults: []time.Duration{time.Second, time.Second, 3 * time.Second},
		},
		{
			name:            "no initial and repeated backoff",
			backoff:         backoff.NewStateful(0, 0, time.Second, 0, 0),
			expectedResults: []time.Duration{time.Second, time.Second, time.Second, time.Second, time.Second},
		},
		{
			name:            "empty backoff",
			backoff:         backoff.NewStateful(0, 0, 0, 0, 0),
			expectedResults: []time.Duration{0, 0, 0, 0, 0},
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			for i, expected := range test.expectedResults {
				actual := test.backoff.Next()
				assert.Equal(t, expected, actual, "invalid %d backoff", i+1)
			}

			test.backoff.Reset()

			for i, expected := range test.expectedResults {
				actual := test.backoff.Next()
				assert.Equal(t, expected, actual, "invalid %d backoff after reset", i+1)
			}
		})
	}
}

func TestStateful_FailShouldReset(t *testing.T) {
	t.Parallel()

	b := backoff.NewStateful(time.Hour, 2*time.Hour, 3*time.Hour, 1, 1)
	assert.False(t, b.Should(), "a backoff that never failed must not hold the caller back")

	b.Fail()
	assert.True(t, b.Should())

	b.Reset()
	assert.False(t, b.Should())
	assert.Equal(t, time.Hour, b.Next(), "the streak restarts at the initial delay after a reset")
	assert.Equal(t, 2*time.Hour, b.Next())
}

func TestTolerantFixed_Should(t *testing.T) {
	t.Parallel()

	b := backoff.NewTolerantFixed(2, time.Hour)

	b.Fail()
	assert.False(t, b.Should(), "the first failure is tolerated")

	b.Fail()
	assert.False(t, b.Should(), "the second failure is tolerated")

	b.Fail()
	assert.True(t, b.Should(), "the third failure imposes the fixed delay")
}
