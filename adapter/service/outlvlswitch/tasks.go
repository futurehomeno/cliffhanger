package outlvlswitch

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/virtualmeter"
	"github.com/futurehomeno/cliffhanger/task"
)

// TaskReporting creates a reporting task.
func TaskReporting(serviceRegistry adapter.ServiceRegistry, frequency time.Duration, voters ...task.Voter) *task.Task {
	voters = append(voters, adapter.IsRegistryInitialized(serviceRegistry))

	return task.New(handleReporting(serviceRegistry), frequency, voters...)
}

// TaskStatePolling creates state polling task adding one default voter.
func TaskStatePolling(serviceRegistry adapter.ServiceRegistry, frequency time.Duration, voters ...task.Voter) *task.Task {
	voters = append(voters, adapter.IsRegistryInitialized(serviceRegistry))

	return task.New(handleStatePolling(serviceRegistry), frequency, voters...)
}

// handleReporting creates handler of a reporting task.
func handleReporting(serviceRegistry adapter.ServiceRegistry) func() {
	return func() {
		for _, s := range serviceRegistry.Services(OutLvlSwitch) {
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

func handleStatePolling(sr adapter.ServiceRegistry) func() {
	return func() {
		for _, s := range sr.Services(OutLvlSwitch) {
			levelSwitch, ok := s.(*service)
			if !ok {
				continue
			}

			thingAddr := levelSwitch.Specification().Address

			if levelSwitch.virtualMeterManager == nil {
				log.Errorf("virtual meter wasn't injected properly, aborting state updates for thing: %s", thingAddr)

				continue
			}

			value, err := levelSwitch.controller.LevelSwitchBinaryStateReport()
			if err != nil {
				log.WithError(err).Errorf("failed to get level switch binary state")

				continue
			}

			mode := virtualmeter.ModeOff
			if value {
				mode = virtualmeter.ModeOn
			}

			level, err := levelSwitch.controller.LevelSwitchLevelReport()
			if err != nil {
				log.WithError(err).Errorf("failed to get level switch level")

				continue
			}

			maxLevel, _ := levelSwitch.Specification().PropertyFloat(PropertyMaxLvl)
			levelNormal := float64(level) / maxLevel

			if err := levelSwitch.virtualMeterManager.Update(thingAddr, mode, levelNormal); err != nil {
				log.WithError(err).Errorf("virtual meter: failed to adjust service state change")
			}
		}
	}
}
