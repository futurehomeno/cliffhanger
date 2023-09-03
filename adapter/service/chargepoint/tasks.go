package chargepoint

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
		for _, s := range serviceRegistry.Services(Chargepoint) {
			chargepoint, ok := s.(Service)
			if !ok {
				continue
			}

			_, err := chargepoint.SendCableLockReport(false)
			if err != nil {
				log.WithError(err).Errorf("adapter: failed to send cable lock report")
			}

			_, err = chargepoint.SendCurrentSessionReport(false)
			if err != nil {
				log.WithError(err).Errorf("adapter: failed to current session report")
			}

			if len(chargepoint.SupportedStates()) > 0 {
				_, err = chargepoint.SendStateReport(false)
				if err != nil {
					log.WithError(err).Errorf("adapter: failed to send chargepoint state report")
				}
			}

			if chargepoint.SupportsAdjustingCurrent() {
				_, err = chargepoint.SendMaxCurrentReport(false)
				if err != nil {
					log.WithError(err).Errorf("adapter: failed to send chargepoint max current report")
				}
			}
		}
	}
}
