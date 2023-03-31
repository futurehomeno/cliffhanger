package outlvlswitch

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
		for _, s := range adapter.Services(OutLvlSwitch) {
			outLvlSwitch, ok := s.(Service)
			if !ok {
				continue
			}

			_, err := outLvlSwitch.SendLevelReport(false)
			if err != nil {
				log.WithError(err).Errorf("adapter: failed to send lvl report")
			}
		}
	}
}
