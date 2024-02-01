package chargepoint

import (
	"strconv"
	"time"
)

// Constants defining service properties and enumerations.
const (
	PropertySupportedStates        = "sup_states"
	PropertySupportedChargingModes = "sup_charging_modes"
	PropertyChargingMode           = "charging_mode"
	PropertyPreviousSession        = "previous_session"
	PropertyStartedAt              = "started_at"
	PropertyFinishedAt             = "finished_at"
	PropertyOfferedCurrent         = "offered_current"
	PropertyCableCurrent           = "cable_current"
	PropertySupportedMaxCurrent    = "sup_max_current"
	PropertySupportedPhaseModes    = "sup_phase_modes"
	PropertySupportedCableLock     = "sup_cable_lock"
	PropertyGridType               = "grid_type"
	PropertyPhases                 = "phases"

	StateDisconnected    State = "disconnected"
	StateRequesting      State = "requesting"
	StateReadyToCharge   State = "ready_to_charge"
	StateCharging        State = "charging"
	StateSwitchingPhases State = "switching_phases"
	StateSuspendedByEVSE State = "suspended_by_evse"
	StateSuspendedByEV   State = "suspended_by_ev"
	StateFinished        State = "finished"
	StateReserved        State = "reserved"
	StateUnavailable     State = "unavailable"
	StateError           State = "error"
	StateUnknown         State = "unknown"

	GridTypeIT GridType = "IT"
	GridTypeTT GridType = "TT"
	GridTypeTN GridType = "TN"

	PhaseModeNL1L2L3 PhaseMode = "NL1L2L3"
	PhaseModeNL1     PhaseMode = "NL1"
	PhaseModeNL2     PhaseMode = "NL2"
	PhaseModeNL3     PhaseMode = "NL3"
	PhaseModeL1L2L3  PhaseMode = "L1L2L3"
	PhaseModeL1L2    PhaseMode = "L1L2"
	PhaseModeL2L3    PhaseMode = "L2L3"
	PhaseModeL3L1    PhaseMode = "L3L1"
)

// State represents a chargepoint state.
type State string

// String returns a string representation of the state.
func (s State) String() string {
	return string(s)
}

// GridType represents a configured grid type.
type GridType string

// String returns a string representation of the grid type.
func (t GridType) String() string {
	return string(t)
}

// PhaseMode represents a configured grid type.
type PhaseMode string

// String returns a string representation of the grid type.
func (t PhaseMode) String() string {
	return string(t)
}

// ChargingSettings represents optional charging settings.
type ChargingSettings struct {
	Mode string
}

// CableReport represents an extended cable status report.
type CableReport struct {
	CableLock    bool
	CableCurrent int64
}

// reportProperties returns a map of report properties.
func (r *CableReport) reportProperties() map[string]string {
	if r.CableCurrent == 0 {
		return nil
	}

	return map[string]string{
		PropertyCableCurrent: strconv.Itoa(int(r.CableCurrent)),
	}
}

// SessionReport represents an extended session report.
type SessionReport struct {
	SessionEnergy         float64
	PreviousSessionEnergy float64
	StartedAt             time.Time
	FinishedAt            time.Time
	OfferedCurrent        int64
}

// reportProperties returns a map of report properties taking into consideration capabilities of the chargepoint.
func (r *SessionReport) reportProperties(supportsAdjustingCurrent bool) map[string]string {
	properties := make(map[string]string)

	if r.PreviousSessionEnergy > 0 {
		properties[PropertyPreviousSession] = strconv.FormatFloat(r.PreviousSessionEnergy, 'f', 2, 64)
	}

	if !r.StartedAt.IsZero() {
		properties[PropertyStartedAt] = r.StartedAt.Format(time.RFC3339)
	}

	if !r.FinishedAt.IsZero() {
		properties[PropertyFinishedAt] = r.FinishedAt.Format(time.RFC3339)
	}

	if supportsAdjustingCurrent {
		properties[PropertyOfferedCurrent] = strconv.Itoa(int(r.OfferedCurrent))
	}

	return properties
}
