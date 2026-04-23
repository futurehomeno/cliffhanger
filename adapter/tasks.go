package adapter

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/task"
)

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

func taskInitialization(adapter Adapter, reportingInterval time.Duration, reportingVoters ...task.Voter) *task.Task {
	return task.New(handleInitialization(adapter), reportingInterval, reportingVoters...)
}

func handleInitialization(adapter Adapter) func() {
	return func() {
		err := adapter.InitializeThings()
		if err != nil {
			log.WithError(err).Errorf("failed to initialize things")
		}
	}
}

func taskConnectivityReporting(adapter Adapter, reportingInterval time.Duration, reportingVoters ...task.Voter) *task.Task {
	return task.New(handleConnectivityReporting(adapter), reportingInterval, reportingVoters...)
}

func handleConnectivityReporting(adapter Adapter) func() {
	return func() {
		for _, t := range adapter.Things() {
			_, err := t.SendConnectivityReport(false)
			if err != nil {
				log.WithError(err).WithField("address", t.Address()).Errorf("failed to send connectivity report")
			}
		}
	}
}

func IsInitialized(adapter Adapter) task.Voter {
	return task.VoterFn(func() bool {
		return adapter.IsInitialized()
	})
}
