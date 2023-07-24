package router

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouter_WithOptions(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name   string
		option Option
		want   *config
	}{
		{
			name:   "With preserved global prefix",
			option: WithPreservedGlobalPrefix(),
			want: &config{
				buffer:               10,
				concurrency:          5,
				preserveGlobalPrefix: true,
			},
		},
		{
			name:   "Sync processing",
			option: WithSyncProcessing(),
			want: &config{
				buffer:      10,
				concurrency: 1,
			},
		},
		{
			name:   "Async processing",
			option: WithAsyncProcessing(3),
			want: &config{
				buffer:      10,
				concurrency: 3,
			},
		},
		{
			name:   "Async processing with incorrect value",
			option: WithAsyncProcessing(-3),
			want: &config{
				buffer:      10,
				concurrency: 5,
			},
		},
		{
			name:   "Message buffer",
			option: WithMessageBuffer(3),
			want: &config{
				buffer:      3,
				concurrency: 5,
			},
		},
		{
			name:   "Message buffer",
			option: WithMessageBuffer(-3),
			want: &config{
				buffer:      10,
				concurrency: 5,
			},
		},
	}

	for _, tc := range tcs {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r, ok := NewRouter(nil, "").(*router)

			assert.True(t, ok)

			r.WithOptions(tc.option)

			assert.Equal(t, tc.want, r.cfg)
		})
	}
}
