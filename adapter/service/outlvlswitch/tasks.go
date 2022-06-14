package outlvlswitch

import (
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/task"

	log "github.com/sirupsen/logrus"
)

// TaskReporting creates a reporting task.
func TaskReporting(adapter adapter.Adapter, frequency time.Duration, voters ...task.Voter) *task.Task {
	return task.New(HandleReporting(adapter), frequency, voters...)
}

// HandleReporting creates handler of a reporting task.
func HandleReporting(adapter adapter.Adapter) func() {
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

			_, err = outLvlSwitch.SendBinaryReport(false)
			if err != nil {
				log.WithError(err).Errorf("adapter: failed to send binary report")
			}
		}
	}
}
