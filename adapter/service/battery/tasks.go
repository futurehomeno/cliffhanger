package battery

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/task"
)

// TaskReporting creates a reporting task.
func TaskReporting(serviceRegistry adapter.ServiceRegistry, frequency time.Duration, voters ...task.Voter) *task.Task {
	voters = append(voters, adapter.IsRegistryInitialized(serviceRegistry))

	return task.New(handleReporting(serviceRegistry), frequency, voters...)
}

// handleReporting creates handler of a reporting task.
func handleReporting(serviceRegistry adapter.ServiceRegistry) func() {
	return func() {
		for _, s := range serviceRegistry.Services(Battery) {
			battery, ok := s.(Service)
			if !ok {
				continue
			}

			if adapter.ShouldSkipServiceTask(serviceRegistry, battery) {
				continue
			}

			_, err := battery.SendBatteryLevelReport(false)
			if err != nil {
				log.WithError(err).Errorf("failed to send battery level report")
			}

			for _, event := range battery.SupportedEvents() {
				_, err = battery.SendBatteryAlarmReport(event, false)
				if err != nil {
					log.WithError(err).Errorf("failed to send battery alarm report for event: %s", event)
				}
			}
		}
	}
}
