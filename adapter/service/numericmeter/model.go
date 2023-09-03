package numericmeter

// Constants defining important properties specific for the service.
const (
	UnitKWh         Unit = "kWh"
	UnitW           Unit = "W"
	UnitA           Unit = "A"
	UnitV           Unit = "V"
	UnitKVAh        Unit = "kVAh"
	UnitHz          Unit = "Hz"
	UnitPowerFactor Unit = "power_factor"
	UnitPulseCount  Unit = "pulse_c"
	UnitCubicMeter  Unit = "cub_m"
	UnitCubicFeet   Unit = "cub_f"
	UnitGallon      Unit = "gallon"

	ValueEnergyImport        Value = "e_import"
	ValueEnergyExport        Value = "e_export"
	ValueLastEnergyExport    Value = "last_e_export"
	ValueLastEnergyImport    Value = "last_e_import"
	ValuePowerImport         Value = "p_import"
	ValueReactivePowerImport Value = "p_import_react"
	ValueApparentPowerImport Value = "p_import_apparent"
	ValueAveragePowerImport  Value = "p_import_avg"
	ValueMinimumPowerImport  Value = "p_import_min"
	ValueMaximumPowerImport  Value = "p_import_max"
	ValuePowerExport         Value = "p_export"
	ValueReactivePowerExport Value = "p_export_react"
	ValueMinimumPowerExport  Value = "p_export_min"
	ValueMaximumPowerExport  Value = "p_export_max"
	ValuePowerFactor         Value = "p_factor"
	ValueFrequency           Value = "freq"
	ValueMinimumFrequency    Value = "freq_min"
	ValueMaximumFrequency    Value = "freq_max"
	ValueVoltagePhase1       Value = "u1"
	ValueVoltagePhase2       Value = "u2"
	ValueVoltagePhase3       Value = "u3"
	ValueCurrentPhase1       Value = "i1"
	ValueCurrentPhase2       Value = "i2"
	ValueCurrentPhase3       Value = "i3"
	ValueDCPower             Value = "dc_p"
	ValueMinimumDCPower      Value = "dc_p_min"
	ValueMaximumDCPower      Value = "dc_p_max"
	ValueDCVoltage           Value = "dc_u"
	ValueMinimumDCVoltage    Value = "dc_u_min"
	ValueMaximumDCVoltage    Value = "dc_u_max"
	ValueDCCurrent           Value = "dc_i"
	ValueMinimumDCCurrent    Value = "dc_i_min"
	ValueMaximumDCCurrent    Value = "dc_i_max"

	PropertyUnit                    = "unit"
	PropertySupportedUnits          = "sup_units"
	PropertySupportedExportUnits    = "sup_export_units"
	PropertySupportedExtendedValues = "sup_extended_vals"
	PropertyIsVirtual               = "is_virtual"
)

// Unit defines metered unit.
type Unit string

// Value defines metered extended value.
type Value string
