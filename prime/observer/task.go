package observer

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/task"
)

// TaskObserver creates tasks for prime observer.
func TaskObserver(observer Observer, refreshInterval time.Duration) []*task.Task {
	return []*task.Task{
		TaskRefreshing(observer, refreshInterval),
	}
}

// TaskRefreshing creates refreshing tasks.
func TaskRefreshing(observer Observer, refreshInterval time.Duration) *task.Task {
	return task.New(HandleRefreshing(observer, false), refreshInterval)
}

// HandleRefreshing creates handler of a refreshing task.
func HandleRefreshing(observer Observer, forceRefresh bool) func() {
	return func() {
		err := observer.Refresh(forceRefresh)
		if err != nil {
			log.WithError(err).Errorf("observer: failed to refresh")
		}
	}
}
