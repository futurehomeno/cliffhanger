package waterheater

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
		for _, s := range serviceRegistry.Services(WaterHeater) {
			waterHeater, ok := s.(Service)
			if !ok {
				continue
			}

			if adapter.ShouldSkipServiceTask(serviceRegistry, waterHeater) {
				continue
			}

			if len(waterHeater.SupportedModes()) > 0 {
				_, err := waterHeater.SendModeReport(false)
				if err != nil {
					log.Errorf("[waterheater] Send mode report. err: %v", err)
				}
			}

			for _, mode := range waterHeater.SupportedSetpoints() {
				_, err := waterHeater.SendSetpointReport(mode, false)
				if err != nil {
					log.Errorf("[waterheater] Send setpoint report. mode: %s err: %v", mode, err)
				}
			}

			if len(waterHeater.SupportedStates()) > 0 {
				_, err := waterHeater.SendStateReport(false)
				if err != nil {
					log.Errorf("[waterheater] Send state report. err: %v", err)
				}
			}
		}
	}
}
