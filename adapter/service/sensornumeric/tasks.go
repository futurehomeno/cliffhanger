package sensornumeric

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

// HandleReporting creates handler of a reporting task.
func HandleReporting(adapter adapter.Adapter) func() {
	return func() {
		for _, s := range adapter.Services("") {
			sensor, ok := s.(Service)
			if !ok {
				continue
			}

			for _, unit := range sensor.SupportedUnits() {
				_, err := sensor.SendReport(unit, false)
				if err != nil {
					log.WithError(err).Errorf("adapter: failed to send sensor report for unit: %s", unit)
				}
			}
		}
	}
}
