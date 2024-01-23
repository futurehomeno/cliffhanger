package virtualmeter

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/task"
)

// Tasks creates tasks for virtual meter that is:
// - reporting task
// - state polling task.
func Tasks(serviceRegistry adapter.ServiceRegistry, reportingInterval, pollingInterval time.Duration, voters ...task.Voter) []*task.Task {
	voters = append(voters, adapter.IsRegistryInitialized(serviceRegistry))

	return task.Combine(
		task.New(handleReporting(serviceRegistry), reportingInterval, voters...),
		task.New(handleStatePolling(serviceRegistry), pollingInterval, voters...),
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
				log.WithError(err).Errorf("task(vms): failed to send reporting interval")
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
				log.WithError(err).Errorf("task(vms): failed to get level switch level")
			}
		}
	}
}
