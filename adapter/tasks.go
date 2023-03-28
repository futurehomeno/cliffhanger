package adapter

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/task"
)

// TaskAdapter creates background tasks specific for an adapter.
func TaskAdapter(
	adapter Adapter,
	reportingInterval time.Duration,
	reportingVoters ...task.Voter,
) []*task.Task {
	return []*task.Task{
		taskConnectivityReporting(adapter, reportingInterval, reportingVoters...),
	}
}

// taskConnectivityReporting creates a reporting task.
func taskConnectivityReporting(adapter Adapter, reportingInterval time.Duration, reportingVoters ...task.Voter) *task.Task {
	return task.New(handleConnectivityReporting(adapter), reportingInterval, reportingVoters...)
}

// handleConnectivityReporting creates handler of a reporting task.
func handleConnectivityReporting(adapter Adapter) func() {
	return func() {
		for _, t := range adapter.Things() {
			_, err := t.SendConnectivityReport(false)
			if err != nil {
				log.WithError(err).WithField("address", t.Address()).Errorf("adapter: failed to send connectivity report")
			}
		}
	}
}
