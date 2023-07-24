package numericmeter

import (
	"strings"
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
		for _, s := range adapter.Services("") {
			if !strings.HasPrefix(s.Name(), prefix) {
				continue
			}

			meter, ok := s.(Service)
			if !ok {
				continue
			}

			handlePeriodicReporting(meter)
		}
	}
}

// handlePeriodicReporting sends all reports for a meter.
func handlePeriodicReporting(meter Service) {
	if meter.SupportsExtendedReport() {
		_, err := meter.SendMeterExtendedReport(meter.SupportedExtendedValues(), false)
		if err != nil {
			log.WithError(err).Errorf("adapter: failed to send meter extended report")
		}

		return
	}

	for _, unit := range meter.SupportedUnits() {
		_, err := meter.SendMeterReport(unit, false)
		if err != nil {
			log.WithError(err).Errorf("adapter: failed to send meter report for unit: %s", unit)
		}
	}

	if !meter.SupportsExportReport() {
		return
	}

	for _, unit := range meter.SupportedExportUnits() {
		_, err := meter.SendMeterExportReport(unit, false)
		if err != nil {
			log.WithError(err).Errorf("adapter: failed to send meter export report for unit: %s", unit)
		}
	}
}