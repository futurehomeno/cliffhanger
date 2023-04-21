package battery

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/task"
)

// TaskReporting creates a reporting task.
func TaskReporting(a adapter.Adapter, frequency time.Duration, voters ...task.Voter) *task.Task {
	voters = append(voters, adapter.IsInitialized(a))

	return task.New(handleReporting(a), frequency, voters...)
}

// handleReporting creates handler of a reporting task.
func handleReporting(adapter adapter.Adapter) func() {
	return func() {
		for _, s := range adapter.Services(Battery) {
			battery, ok := s.(Service)
			if !ok {
				continue
			}

			_, err := battery.SendBatteryLevelReport(false)
			if err != nil {
				log.WithError(err).Errorf("adapter: failed to send battery level report")
			}

			for _, event := range battery.SupportedEvents() {
				_, err = battery.SendBatteryAlarmReport(event, false)
				if err != nil {
					log.WithError(err).Errorf("adapter: failed to send battery alarm report for event: %s", event)
				}
			}
		}
	}
}
