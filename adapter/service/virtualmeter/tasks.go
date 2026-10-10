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
func Tasks(
	serviceRegistry adapter.ServiceRegistry,
	mr Manager,
	reportingInterval,
	pollingInterval,
	cleaningInterval time.Duration,
	voters ...task.Voter,
) []*task.Task {
	voters = append(voters, adapter.IsRegistryInitialized(serviceRegistry))

	return task.Combine(
		task.New(handleReporting(serviceRegistry), reportingInterval, voters...),
		outlvlswitch.TaskReporting(serviceRegistry, pollingInterval, voters...),
		task.New(handleGarbageCleaning(serviceRegistry, mr), cleaningInterval, voters...),
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
				log.Errorf("[cliff] Send modes report. err: %v", err)
			}
		}
	}
}

func handleGarbageCleaning(sr adapter.ServiceRegistry, mr Manager) func() {
	m, ok := mr.(*manager)
	if !ok {
		log.Errorf("[cliff] Manager cast failed")

		return func() {}
	}

	return func() {
		liveTopics := make(map[string]struct{})

		for _, s := range sr.Services(VirtualMeterElec) {
			if vmeter, ok := s.(Service); ok {
				liveTopics[vmeter.Topic()] = struct{}{}
			}
		}

		if err := m.cleanOrphanedDevices(liveTopics); err != nil {
			log.Errorf("[cliff] Clean orphaned devices. err: %v", err)
		}
	}
}
