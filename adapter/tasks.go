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
	reportingVoters = append(reportingVoters, IsInitialized(adapter))

	return []*task.Task{
		taskInitialization(adapter, reportingInterval, task.WhenNot(IsInitialized(adapter))),
		taskConnectivityReporting(adapter, reportingInterval, reportingVoters...),
	}
}

// taskInitialization creates an initialization task.
func taskInitialization(adapter Adapter, reportingInterval time.Duration, reportingVoters ...task.Voter) *task.Task {
	return task.New(handleInitialization(adapter), reportingInterval, reportingVoters...)
}

// handleInitialization creates handler of an initialization task.
func handleInitialization(adapter Adapter) func() {
	return func() {
		err := adapter.InitializeThings()
		if err != nil {
			log.WithError(err).Errorf("adapter: failed to initialize things")
		}
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

// IsInitialized returns a voter that checks if the adapter is initialized.
func IsInitialized(adapter Adapter) task.Voter {
	return task.VoterFn(func() bool {
		return adapter.IsInitialized()
	})
}
