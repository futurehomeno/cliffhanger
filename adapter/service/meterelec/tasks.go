package meterelec

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
		for _, s := range adapter.Services(MeterElec) {
			meterElec, ok := s.(Service)
			if !ok {
				continue
			}

			if meterElec.SupportsExtendedReport() {
				_, err := meterElec.SendMeterExtendedReport(false)
				if err != nil {
					log.WithError(err).Errorf("adapter: failed to send meter extended report")
				}

				continue
			}

			for _, unit := range meterElec.SupportedUnits() {
				_, err := meterElec.SendMeterReport(unit, false)
				if err != nil {
					log.WithError(err).Errorf("adapter: failed to send meter report for unit: %s", unit)
				}
			}
		}
	}
}
