package chargepoint

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/task"
)

func TaskReporting(serviceRegistry adapter.ServiceRegistry, frequency time.Duration, voters ...task.Voter) *task.Task {
	voters = append(voters, adapter.IsRegistryInitialized(serviceRegistry))

	return task.New(handleReporting(serviceRegistry), frequency, voters...)
}

func handleReporting(serviceRegistry adapter.ServiceRegistry) func() {
	return func() {
		for _, s := range serviceRegistry.Services(Chargepoint) {
			chargepoint, ok := s.(Service)
			if !ok {
				continue
			}

			if adapter.ShouldSkipServiceTask(serviceRegistry, chargepoint) {
				continue
			}

			sendChargepointReports(chargepoint)
		}
	}
}

func sendChargepointReports(s Service) {
	_, err := s.SendCurrentSessionReport(false)
	if adapter.ReportFailed(err) {
		log.Errorf("[chargepoint] Send current session report. err: %v", err)
	}

	if len(s.SupportedStates()) > 0 {
		_, err = s.SendStateReport(false)
		if adapter.ReportFailed(err) {
			log.Errorf("[chargepoint] Send state report. err: %v", err)
		}
	}

	if s.SupportsAdjustingMaxCurrent() {
		_, err = s.SendMaxCurrentReport(false)
		if adapter.ReportFailed(err) {
			log.Errorf("[chargepoint] Send max current report. err: %v", err)
		}
	}

	if s.IsPhaseModeAware() {
		_, err = s.SendPhaseModeReport(false)
		if adapter.ReportFailed(err) {
			log.Errorf("[chargepoint] Send phase mode report. err: %v", err)
		}
	}

	if s.IsCableLockAware() {
		_, err := s.SendCableLockReport(false)
		if adapter.ReportFailed(err) {
			log.Errorf("[chargepoint] Send cable lock report. err: %v", err)
		}
	}
}
