package observer

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/task"
)

// TaskRefreshing creates refreshing tasks.
func TaskRefreshing(observer Observer, forcedRefreshInterval time.Duration) *task.Task {
	return task.New(HandleRefreshing(observer, true), forcedRefreshInterval)
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
