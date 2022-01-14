package han

import (
	"github.com/futurehomeno/cliffhanger/adapter"
)

// HAN is an interface representing a HAN meter.
type HAN interface {
	adapter.Thing

	// Report returns simplified HAN meter report based on input unit.
	Report(unit string) (float64, error)
	// ExtendedReport returns extended HAN meter report. Should return nil if extended report is not supported.
	ExtendedReport() (map[string]float64, error)
	// SupportedUnits returns units that are supported by the simplified meter report.
	SupportedUnits() []string
	// SupportedExtendedValues returns extended values that are supported by the extended meter report.
	SupportedExtendedValues() []string
	// SupportsExtendedReport returns true if meter supports the extended report.
	SupportsExtendedReport() bool
}
