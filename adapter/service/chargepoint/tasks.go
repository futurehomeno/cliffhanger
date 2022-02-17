package chargepoint

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
		for _, s := range adapter.Services(Chargepoint) {
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
		}
	}
}
