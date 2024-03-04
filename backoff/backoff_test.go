package backoff_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/backoff"
)

func TestBackoff_ShouldBackoff(t *testing.T) {
	t.Parallel()

	now := time.Now()

	testCases := []struct {
		name      string
		lastErrAt time.Time
		failures  uint32
		want      bool
	}{
		{
			name: "check for the first time: returns false",
			want: false,
		},
		{
			name:      "initial threshold: last failure was 3 seconds ago: returns true",
			lastErrAt: now.Add(-3 * time.Second),
			failures:  1,
			want:      true,
		},
		{
			name:      "initial threshold: last failure was 8 seconds ago: returns false",
			lastErrAt: now.Add(-8 * time.Second),
			failures:  1,
			want:      false,
		},
		{
			name:      "repeated threshold: last failure was 13 minutes ago: returns true",
			lastErrAt: now.Add(-13 * time.Minute),
			failures:  4,
			want:      true,
		},
		{
			name:      "repeated threshold: last failure was 16 minutes ago: returns false",
			lastErrAt: now.Add(-16 * time.Minute),
			failures:  4,
			want:      false,
		},
		{
			name:      "final threshold: last failure was 20 hours ago: returns true",
			lastErrAt: now.Add(-20 * time.Hour),
			failures:  10,
			want:      true,
		},
		{
			name:      "final threshold: last failure was 26 hours ago: returns false",
			lastErrAt: now.Add(-26 * time.Hour),
			failures:  10,
			want:      false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b := backoff.New(5*time.Second, 15*time.Minute, 24*time.Hour, 3, 3)

			got := b.Should(tc.lastErrAt, tc.failures)

			assert.Equal(t, tc.want, got)
		})
	}
}
