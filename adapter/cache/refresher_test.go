package cache_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

func TestRefresher_Refresh(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name              string
		refreshMock       *refreshMock
		refresherInterval time.Duration
		refresherOptions  []cache.RefresherOption
		want              any
		wantErr           bool
		wantFailing       bool
		repeatCount       int
		repeatInterval    time.Duration
		resetAt           int
	}{
		{
			name:              "Single successful refresh",
			refreshMock:       newRefreshMock().mockRefresh("test", nil, false),
			refresherInterval: 25 * time.Millisecond,
			refresherOptions:  []cache.RefresherOption{cache.WithDefaultOptions()},
			want:              "test",
			wantFailing:       false,
			wantErr:           false,
		},
		{
			name:              "Single failed refresh",
			refreshMock:       newRefreshMock().mockRefresh(nil, errors.New("test"), false),
			refresherInterval: 25 * time.Millisecond,
			refresherOptions:  []cache.RefresherOption{cache.WithFailureThreshold(1)},
			want:              nil,
			wantFailing:       false,
			wantErr:           true,
		},
		{
			name:              "Two failed refreshes",
			refreshMock:       newRefreshMock().mockRefresh(nil, errors.New("test"), false),
			refresherInterval: 25 * time.Millisecond,
			refresherOptions:  []cache.RefresherOption{cache.WithFailureThreshold(1)},
			want:              nil,
			wantFailing:       true,
			wantErr:           true,
			repeatCount:       1,
			repeatInterval:    30 * time.Millisecond,
		},
		{
			name:              "Single successful refresh followed by cached value",
			refreshMock:       newRefreshMock().mockRefresh("test", nil, true),
			refresherInterval: 50 * time.Millisecond,
			refresherOptions:  []cache.RefresherOption{cache.WithDefaultOptions()},
			want:              "test",
			wantFailing:       false,
			wantErr:           false,
			repeatCount:       1,
			repeatInterval:    30 * time.Millisecond,
		},
		{
			name: "Failed refresh with all backoff thresholds finished with successful refresh",
			refreshMock: newRefreshMock().
				mockRefresh(nil, errors.New("test"), true).
				mockRefresh(nil, errors.New("test"), true).
				mockRefresh(nil, errors.New("test"), true).
				mockRefresh("test", nil, true),
			refresherInterval: 25 * time.Millisecond,
			refresherOptions:  []cache.RefresherOption{cache.WithBackoff(15*time.Millisecond, 25*time.Millisecond, 35*time.Millisecond, 1)},
			want:              "test",
			wantFailing:       false,
			wantErr:           false,
			repeatCount:       10,
			repeatInterval:    10 * time.Millisecond,
		},
		{
			name: "Successful refresh after reset",
			refreshMock: newRefreshMock().
				mockRefresh("test", nil, true).
				mockRefresh("test", nil, true),
			refresherInterval: 50 * time.Millisecond,
			refresherOptions:  []cache.RefresherOption{cache.WithBackoff(15*time.Millisecond, 25*time.Millisecond, 35*time.Millisecond, 1)},
			want:              "test",
			wantFailing:       false,
			wantErr:           false,
			repeatCount:       3,
			repeatInterval:    10 * time.Millisecond,
			resetAt:           1,
		},
	}

	for _, tc := range tcs {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			refresher := cache.NewRefresher(tc.refreshMock.refresh, tc.refresherInterval, tc.refresherOptions...)

			var (
				got any
				err error
			)

			for i := 0; i < tc.repeatCount+1; i++ {
				if i == tc.resetAt {
					refresher.Reset()
				}

				got, err = refresher.Refresh()

				time.Sleep(tc.repeatInterval)
			}

			assert.Equal(t, tc.wantFailing, refresher.IsFailing())

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.want, got)

			tc.refreshMock.AssertExpectations(t)
		})
	}
}

func newRefreshMock() *refreshMock {
	return &refreshMock{}
}

type refreshMock struct {
	mock.Mock
}

func (m *refreshMock) refresh() (any, error) {
	args := m.Called()

	return args.Get(0), args.Error(1)
}

func (m *refreshMock) mockRefresh(want any, err error, once bool) *refreshMock {
	c := m.On("refresh").Return(want, err)

	if once {
		c.Once()
	}

	return m
}
