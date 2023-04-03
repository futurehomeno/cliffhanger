package waterheater

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
		for _, s := range adapter.Services(WaterHeater) {
			waterHeater, ok := s.(Service)
			if !ok {
				continue
			}

			if len(waterHeater.SupportedModes()) > 0 {
				_, err := waterHeater.SendModeReport(false)
				if err != nil {
					log.WithError(err).Errorf("adapter: failed to send water heater mode report")
				}
			}

			for _, mode := range waterHeater.SupportedSetpoints() {
				_, err := waterHeater.SendSetpointReport(mode, false)
				if err != nil {
					log.WithError(err).Errorf("adapter: failed to send water heater setpoint report for mode %s", mode)
				}
			}

			if len(waterHeater.SupportedStates()) > 0 {
				_, err := waterHeater.SendStateReport(false)
				if err != nil {
					log.WithError(err).Errorf("adapter: failed to send water heater state report")
				}
			}
		}
	}
}
