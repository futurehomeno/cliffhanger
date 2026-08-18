package thermostat

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
		for _, s := range serviceRegistry.Services(Thermostat) {
			thermostat, ok := s.(Service)
			if !ok {
				continue
			}

			if adapter.ShouldSkipServiceTask(serviceRegistry, thermostat) {
				continue
			}

			if len(thermostat.SupportedModes()) > 0 {
				_, err := thermostat.SendModeReport(false)
				if err != nil {
					log.Errorf("[thermostat] Send mode report. err: %v", err)
				}
			}

			for _, mode := range thermostat.SupportedSetpoints() {
				_, err := thermostat.SendSetpointReport(mode, false)
				if err != nil {
					log.Errorf("[thermostat] Send setpoint report. mode: %s err: %v", mode, err)
				}
			}

			if len(thermostat.SupportedStates()) > 0 {
				_, err := thermostat.SendStateReport(false)
				if err != nil {
					log.Errorf("[thermostat] Send state report. err: %v", err)
				}
			}
		}
	}
}
