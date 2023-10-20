package numericmeter

// Constants defining important properties specific for the service.
const (
	UnitKWh         Unit = "kWh"
	UnitW           Unit = "W"
	UnitA           Unit = "A"
	UnitV           Unit = "V"
	UnitVA          Unit = "VA"
	UnitKVAh        Unit = "kVAh"
	UnitVAr         Unit = "VAr"
	UnitKVArh       Unit = "kVArh"
	UnitHz          Unit = "Hz"
	UnitPowerFactor Unit = "power_factor"
	UnitPulseCount  Unit = "pulse_c"
	UnitCubicMeter  Unit = "cub_m"
	UnitCubicFeet   Unit = "cub_f"
	UnitGallon      Unit = "gallon"

	ValuePowerImport               Value = "p_import"
	ValuePowerImportPhase1         Value = "p1"
	ValuePowerImportPhase2         Value = "p2"
	ValuePowerImportPhase3         Value = "p3"
	ValuePowerExport               Value = "p_export"
	ValuePowerExportPhase1         Value = "p1_export"
	ValuePowerExportPhase2         Value = "p2_export"
	ValuePowerExportPhase3         Value = "p3_export"
	ValueEnergyImport              Value = "e_import"
	ValueEnergyImportPhase1        Value = "e1_import"
	ValueEnergyImportPhase2        Value = "e2_import"
	ValueEnergyImportPhase3        Value = "e3_import"
	ValueEnergyExport              Value = "e_export"
	ValueEnergyExportPhase1        Value = "e1_export"
	ValueEnergyExportPhase2        Value = "e2_export"
	ValueEnergyExportPhase3        Value = "e3_export"
	ValueReactiveEnergyImport      Value = "e_import_react"
	ValueApparentEnergyImport      Value = "e_import_apparent"
	ValueReactiveEnergyExport      Value = "e_export_react"
	ValueApparentEnergyExport      Value = "e_export_apparent"
	ValueReactivePowerImport       Value = "p_import_react"
	ValueReactivePowerImportPhase1 Value = "p1_import_react"
	ValueReactivePowerImportPhase2 Value = "p2_import_react"
	ValueReactivePowerImportPhase3 Value = "p3_import_react"
	ValueReactivePowerExport       Value = "p_export_react"
	ValueReactivePowerExportPhase1 Value = "p1_export_react"
	ValueReactivePowerExportPhase2 Value = "p2_export_react"
	ValueReactivePowerExportPhase3 Value = "p3_export_react"
	ValueApparentPowerImport       Value = "p_import_apparent"
	ValueApparentPowerImportPhase1 Value = "p1_import_apparent"
	ValueApparentPowerImportPhase2 Value = "p2_import_apparent"
	ValueApparentPowerImportPhase3 Value = "p3_import_apparent"
	ValueApparentPowerExport       Value = "p_export_apparent"
	ValueApparentPowerExportPhase1 Value = "p1_export_apparent"
	ValueApparentPowerExportPhase2 Value = "p2_export_apparent"
	ValueApparentPowerExportPhase3 Value = "p3_export_apparent"
	ValueVoltage                   Value = "u"
	ValueVoltagePhase1             Value = "u1"
	ValueVoltagePhase2             Value = "u2"
	ValueVoltagePhase3             Value = "u3"
	ValueVoltageExport             Value = "u_export"
	ValueVoltageExportPhase1       Value = "u1_export"
	ValueVoltageExportPhase2       Value = "u2_export"
	ValueVoltageExportPhase3       Value = "u3_export"
	ValueCurrent                   Value = "i"
	ValueCurrentPhase1             Value = "i1"
	ValueCurrentPhase2             Value = "i2"
	ValueCurrentPhase3             Value = "i3"
	ValueCurrentExport             Value = "i_export"
	ValueCurrentExportPhase1       Value = "i1_export"
	ValueCurrentExportPhase2       Value = "i2_export"
	ValueCurrentExportPhase3       Value = "i3_export"
	ValuePowerFactor               Value = "p_factor"
	ValuePowerFactorPhase1         Value = "p1_factor"
	ValuePowerFactorPhase2         Value = "p2_factor"
	ValuePowerFactorPhase3         Value = "p3_factor"
	ValuePowerFactorExport         Value = "p_factor_export"
	ValuePowerFactorExportPhase1   Value = "p1_factor_export"
	ValuePowerFactorExportPhase2   Value = "p2_factor_export"
	ValuePowerFactorExportPhase3   Value = "p3_factor_export"
	ValueFrequency                 Value = "freq"
	ValueDCPower                   Value = "dc_p"
	ValueDCVoltage                 Value = "dc_u"
	ValueDCCurrent                 Value = "dc_i"

	PropertyUnit                    = "unit"
	PropertySupportedUnits          = "sup_units"
	PropertySupportedExportUnits    = "sup_export_units"
	PropertySupportedExtendedValues = "sup_extended_vals"
	PropertyIsVirtual               = "is_virtual"
)

// Units is a collection of units.
type Units []Unit

// Strings returns slice of string representation of the units.
func (u Units) Strings() []string {
	output := make([]string, len(u))

	for i, v := range u {
		output[i] = v.String()
	}

	return output
}

// NewUnits creates a new collection of units.
func NewUnits[T string | Unit](input ...T) Units {
	output := make(Units, len(input))

	for i, v := range input {
		output[i] = Unit(v)
	}

	return output
}

// Unit defines metered unit.
type Unit string

// String returns string representation of the unit.
func (u Unit) String() string {
	return string(u)
}

// ValuesReport is a collection of extended values report.
type ValuesReport map[Value]float64

// Map returns float map representation of the values report.
func (r ValuesReport) Map() map[string]float64 {
	output := make(map[string]float64, len(r))

	for k, v := range r {
		output[k.String()] = v
	}

	return output
}

// Values is a collection of extended values.
type Values []Value

// Strings returns slice of string representation of the values.
func (v Values) Strings() []string {
	output := make([]string, len(v))

	for i, val := range v {
		output[i] = val.String()
	}

	return output
}

// NewValues creates a new collection of extended values.
func NewValues[T string | Value](input ...T) Values {
	output := make(Values, len(input))

	for i, v := range input {
		output[i] = Value(v)
	}

	return output
}

// Value defines metered extended value.
type Value string

// String returns string representation of the value.
func (v Value) String() string {
	return string(v)
}
