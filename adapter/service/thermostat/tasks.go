package thermostat

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
		for _, s := range adapter.Services(Thermostat) {
			thermostat, ok := s.(Service)
			if !ok {
				continue
			}

			if len(thermostat.SupportedModes()) > 0 {
				_, err := thermostat.SendModeReport(false)
				if err != nil {
					log.WithError(err).Errorf("adapter: failed to send thermostat mode report")
				}
			}

			for _, mode := range thermostat.SupportedSetpoints() {
				_, err := thermostat.SendSetpointReport(mode, false)
				if err != nil {
					log.WithError(err).Errorf("adapter: failed to send thermostat setpoint report for mode %s", mode)
				}
			}

			if len(thermostat.SupportedStates()) > 0 {
				_, err := thermostat.SendStateReport(false)
				if err != nil {
					log.WithError(err).Errorf("adapter: failed to send thermostat state report")
				}
			}
		}
	}
}
