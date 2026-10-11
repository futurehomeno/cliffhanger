package cache_test

import (
	"errors"
	"sync"
	"sync/atomic"
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

func TestRefresher_BackoffWrapsLastError(t *testing.T) {
	t.Parallel()

	cause := errors.New("myuplink: 503")
	refresher := cache.NewRefresher(newRefreshMock().mockRefresh(nil, cause, true).refresh, time.Minute, cache.WithDefaultBackoff())

	_, err := refresher.Refresh()
	assert.ErrorIs(t, err, cause)

	_, err = refresher.Refresh()
	assert.ErrorIs(t, err, cache.ErrBackoff)
	assert.ErrorIs(t, err, cause)
	assert.ErrorContains(t, err, "refresher: backoff is in effect: last error: myuplink: 503")
}

func TestRefresher_ResetKeepsBackoff(t *testing.T) {
	t.Parallel()

	calls := 0
	refresher := cache.NewRefresher(func() (int, error) {
		calls++

		if calls == 2 {
			return 0, errors.New("cloud down")
		}

		return calls, nil
	}, time.Nanosecond, cache.WithDefaultBackoff())

	_, err := refresher.Refresh()
	assert.NoError(t, err)

	time.Sleep(time.Millisecond)

	_, err = refresher.Refresh()
	assert.Error(t, err)

	refresher.Reset()

	_, err = refresher.Refresh()
	assert.ErrorIs(t, err, cache.ErrBackoff, "a reset must not lift the backoff")
	assert.Equal(t, 2, calls)
}

func TestRefresher_NoBackoffWithoutOption(t *testing.T) {
	t.Parallel()

	m := newRefreshMock().mockRefresh(nil, errors.New("test"), false)
	refresher := cache.NewRefresher(m.refresh, time.Minute)

	_, err := refresher.Refresh()
	assert.Error(t, err)

	_, err = refresher.Refresh()
	assert.Error(t, err)
	assert.NotErrorIs(t, err, cache.ErrBackoff)
	m.AssertNumberOfCalls(t, "refresh", 2)
	assert.True(t, refresher.IsFailing(), "the failure threshold defaults to zero")
}

func TestRefresher_ConcurrentCallersShareOneRefresh(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	release := make(chan struct{})
	refresher := cache.NewRefresher(func() (int, error) {
		calls.Add(1)
		<-release

		return 42, nil
	}, time.Minute)

	const callers = 5

	var wg sync.WaitGroup

	results := make(chan int, callers)

	for range callers {
		wg.Go(func() {
			v, err := refresher.Refresh()
			assert.NoError(t, err)

			results <- v
		})
	}

	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()
	close(results)

	assert.Equal(t, int32(1), calls.Load(), "concurrent callers must not refresh in parallel")

	for v := range results {
		assert.Equal(t, 42, v)
	}
}
