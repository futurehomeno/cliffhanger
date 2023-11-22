package numericmeter

import (
	"github.com/futurehomeno/cliffhanger/adapter/service/virtualmeter"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/task"
)

// TaskReporting creates a reporting task.
func TaskReporting(serviceRegistry adapter.ServiceRegistry, frequency time.Duration, voters ...task.Voter) *task.Task {
	voters = append(voters, adapter.IsRegistryInitialized(serviceRegistry))

	return task.New(handleReporting(serviceRegistry), frequency, voters...)
}

// TaskVirtualReporting creates a reporting task for a virtual meter.
func TaskVirtualReporting(serviceRegistry adapter.ServiceRegistry, vmeterManager virtualmeter.VirtualMeterManager, voters ...task.Voter) *task.Task {
	voters = append(voters, adapter.IsRegistryInitialized(serviceRegistry))

	return task.NewNamedTask(virtualmeter.TaskVirtualReporter, handleVirtualReporting(serviceRegistry), vmeterManager.ReportingInterval(), voters...)
}

// handleReporting creates handler of a reporting task.
func handleReporting(serviceRegistry adapter.ServiceRegistry) func() {
	return func() {
		for _, s := range serviceRegistry.Services("") {
			if !strings.HasPrefix(s.Name(), prefix) {
				continue
			}

			meter, ok := s.(Service)
			if !ok {
				continue
			}

			if meter.Specification().PropertyBool(PropertyIsVirtual) {
				continue
			}

			handlePeriodicReporting(meter)
		}
	}
}

// handleVirtualReporting creates handler of a reporting task, but only for virtual meters.
func handleVirtualReporting(serviceRegistry adapter.ServiceRegistry) func() {
	return func() {
		for _, s := range serviceRegistry.Services("") {
			if !strings.HasPrefix(s.Name(), prefix) {
				continue
			}

			meter, ok := s.(Service)
			if !ok {
				continue
			}

			if !meter.Specification().PropertyBool(PropertyIsVirtual) {
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
