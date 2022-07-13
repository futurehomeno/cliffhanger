package battery

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/task"
)

// TaskReporting creates a reporting task.
func TaskReporting(adapter adapter.Adapter, frequency time.Duration, voters ...task.Voter) *task.Task {
	return task.New(HandleReporting(adapter), frequency, voters...)
}

// HandleReporting
func HandleReporting(adapter adapter.Adapter) func() {
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

			_, err = battery.SendBatteryAlarmReport(false)
			if err != nil {
				log.WithError(err).Errorf("adapter: failed to send battery alarm report")
			}

			_, err = battery.SendBatteryHealthReport(false)
			if err != nil {
				log.WithError(err).Errorf("adapter: failed to send battery health report")
			}

			_, err = battery.SendBatterySensorReport(false)
			if err != nil {
				log.WithError(err).Errorf("adapter: failed to send battery sensor report")
			}

			_, err = battery.SendBatteryFullReport(false)
			if err != nil {
				log.WithError(err).Errorf("adapter: failed to send battery full report")
			}
		}
	}
}
