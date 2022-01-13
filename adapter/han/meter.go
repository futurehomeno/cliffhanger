package han

type Meter interface {
	// GetReport returns simplified meter report based on input unit.
	GetReport(unit string) (float64, error)
	// GetExtendedReport returns extended meter report. Should return nil if extended report is not supported.
	GetExtendedReport() (map[string]float64, error)
	// GetSupportedUnits returns units that are supported by the simplified meter report.
	GetSupportedUnits() []string
	// GetSupportedExtendedValues returns extended values that are supported by the extended meter report.
	GetSupportedExtendedValues() []string
	// SupportsExtendedReport returns true if meter supports the extended report.
	SupportsExtendedReport() bool
}



