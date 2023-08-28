package numericsensor

import (
	"strings"
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
		for _, s := range serviceRegistry.Services("") {
			if !strings.HasPrefix(s.Name(), prefix) {
				continue
			}

			sensor, ok := s.(Service)
			if !ok {
				continue
			}

			for _, unit := range sensor.SupportedUnits() {
				_, err := sensor.SendSensorReport(unit, false)
				if err != nil {
					log.WithError(err).Errorf("adapter: failed to send sensor report for unit: %s", unit)
				}
			}
		}
	}
}
