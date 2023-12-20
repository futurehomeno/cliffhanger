package virtualmeter

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/task"
)

// TaskStatePolling creates state polling task adding one default voter.
func TaskReporting(serviceRegistry adapter.ServiceRegistry, frequency time.Duration, voters ...task.Voter) []*task.Task {
	voters = append(voters, adapter.IsRegistryInitialized(serviceRegistry))

	return task.Combine(
		task.New(handleReporting(serviceRegistry), frequency, voters...),
		task.New(handleStatePolling(serviceRegistry), frequency, voters...),
	)
}

// handleReporting creates handler of a reporting task.
func handleReporting(serviceRegistry adapter.ServiceRegistry) func() {
	return func() {
		for _, s := range serviceRegistry.Services(VirtualMeterElec) {
			vmeter, ok := s.(Service)
			if !ok {
				continue
			}

			if _, err := vmeter.SendModesReport(false); err != nil {
				log.WithError(err).Errorf("adapter: failed to send reporting interval")
			}
		}
	}
}

// handleStatePolling uses controller to get the level and the current mode of the device to update the virtual meter manager.
func handleStatePolling(sr adapter.ServiceRegistry) func() {
	return func() {
		for _, s := range sr.Services(outlvlswitch.OutLvlSwitch) {
			levelSwitch, ok := s.(outlvlswitch.Service)
			if !ok {
				continue
			}

			_, err := levelSwitch.SendLevelReport(false)
			if err != nil {
				log.WithError(err).Errorf("failed to get level switch level")
			}
		}
	}
}
