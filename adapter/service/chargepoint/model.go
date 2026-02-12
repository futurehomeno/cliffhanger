package chargepoint

import (
	"slices"
	"strconv"
	"time"

	"github.com/futurehomeno/galvanize/v2/unit"
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

	GridTypeIT      GridType = "IT"
	GridTypeTT      GridType = "TT"
	GridTypeTN      GridType = "TN"
	GridTypeUnknown GridType = ""

	PhaseModeNL1L2L3 PhaseMode = "NL1L2L3"
	PhaseModeNL1L2   PhaseMode = "NL1L2"
	PhaseModeNL2L3   PhaseMode = "NL2L3"
	PhaseModeNL1     PhaseMode = "NL1"
	PhaseModeNL2     PhaseMode = "NL2"
	PhaseModeNL3     PhaseMode = "NL3"
	PhaseModeL1L2L3  PhaseMode = "L1L2L3"
	PhaseModeL1L2    PhaseMode = "L1L2"
	PhaseModeL2L3    PhaseMode = "L2L3"
	PhaseModeL3L1    PhaseMode = "L3L1"
	PhaseModeUnknown PhaseMode = ""
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
func (t GridType) Str() string {
	return string(t)
}

// PhaseMode represents a configured grid type.
type PhaseMode string

// String returns a string representation of the grid type.
func (t PhaseMode) Str() string {
	return string(t)
}

// PhaseModeToPhases is a helper function which translates a particular phase mode to a list of phases it utilizes.
func (phaseMode PhaseMode) PhasesList() []unit.Phase {
	switch phaseMode { //nolint:exhaustive
	case PhaseModeNL1L2L3:
		return []unit.Phase{unit.PhaseL1, unit.PhaseL2, unit.PhaseL3}
	case PhaseModeNL1:
		return []unit.Phase{unit.PhaseL1}
	case PhaseModeNL2:
		return []unit.Phase{unit.PhaseL2}
	case PhaseModeNL3:
		return []unit.Phase{unit.PhaseL3}
	case PhaseModeL1L2L3:
		return []unit.Phase{unit.PhaseL1, unit.PhaseL2, unit.PhaseL3}
	case PhaseModeL1L2:
		return []unit.Phase{unit.PhaseL1, unit.PhaseL2}
	case PhaseModeL2L3:
		return []unit.Phase{unit.PhaseL2, unit.PhaseL3}
	case PhaseModeL3L1:
		return []unit.Phase{unit.PhaseL1, unit.PhaseL3}
	default:
		return []unit.Phase{unit.PhaseL1, unit.PhaseL2, unit.PhaseL3}
	}
}

// AllPhaseModes is a helper function which returns all possible phase modes.
func AllPhaseModes() []PhaseMode {
	return []PhaseMode{
		PhaseModeNL1L2L3,
		PhaseModeNL1,
		PhaseModeNL2,
		PhaseModeNL3,
		PhaseModeL1L2L3,
		PhaseModeL1L2,
		PhaseModeL2L3,
		PhaseModeL3L1,
	}
}

// AllConnectionEarthingsTypes is a helper function which returns all possible grid earthing types.
func AllConnectionEarthingsTypes() []GridType {
	return []GridType{
		GridTypeTN,
		GridTypeTT,
		GridTypeIT,
	}
}

// PhasesToPhaseMode is a helper function which translates a list of phases to a particular phase mode utilizing it in a provided grid earthing type.
func PhasesToPhaseMode(earthingType GridType, phases ...unit.Phase) PhaseMode { //nolint:cyclop
	if earthingType == GridTypeTN {
		if len(phases) == 3 && slices.Contains(phases, unit.PhaseL1) && slices.Contains(phases, unit.PhaseL2) && slices.Contains(phases, unit.PhaseL3) {
			return PhaseModeNL1L2L3
		}

		if len(phases) == 1 && phases[0] == unit.PhaseL1 {
			return PhaseModeNL1
		}

		if len(phases) == 1 && phases[0] == unit.PhaseL2 {
			return PhaseModeNL2
		}

		if len(phases) == 1 && phases[0] == unit.PhaseL3 {
			return PhaseModeNL3
		}
	}

	if earthingType == GridTypeIT || earthingType == GridTypeTT {
		if len(phases) == 3 && slices.Contains(phases, unit.PhaseL1) && slices.Contains(phases, unit.PhaseL2) && slices.Contains(phases, unit.PhaseL3) {
			return PhaseModeL1L2L3
		}

		if len(phases) == 2 && slices.Contains(phases, unit.PhaseL1) && slices.Contains(phases, unit.PhaseL2) {
			return PhaseModeL1L2
		}

		if len(phases) == 2 && slices.Contains(phases, unit.PhaseL2) && slices.Contains(phases, unit.PhaseL3) {
			return PhaseModeL2L3
		}

		if len(phases) == 2 && slices.Contains(phases, unit.PhaseL3) && slices.Contains(phases, unit.PhaseL1) {
			return PhaseModeL3L1
		}
	}

	return PhaseModeUnknown
}

// some phase modes are forbidden - not tested or other problems (AMS does not report i2)
func AvailablePhaseModes(earthingType GridType, supportedPhaseModes []PhaseMode, utilizedPhases int) []PhaseMode {
	if len(supportedPhaseModes) == 0 {
		return []PhaseMode{PhaseModeUnknown}
	}

	if earthingType == GridTypeIT || earthingType == GridTypeTT {
		ret := []PhaseMode{PhaseModeL3L1}

		if !slices.Contains(supportedPhaseModes, ret[0]) {
			return []PhaseMode{PhaseModeUnknown}
		}

		return ret
	}

	if utilizedPhases == 3 {
		if !slices.Contains(supportedPhaseModes, PhaseModeNL1L2L3) {
			return []PhaseMode{PhaseModeUnknown}
		}

		return []PhaseMode{PhaseModeNL1L2L3}
	}

	ret := []PhaseMode{}

	for _, pm := range []PhaseMode{PhaseModeNL1, PhaseModeNL2, PhaseModeNL3} {
		if slices.Contains(supportedPhaseModes, pm) {
			ret = append(ret, pm)
		}
	}

	if len(ret) == 0 {
		return []PhaseMode{PhaseModeUnknown}
	}

	return ret
}

// ChargingSettings represents optional charging settings.
type ChargingSettings struct {
	Mode string
}

// CableReport represents an extended cable status report.
type CableReport struct {
	CableLock    bool
	CableCurrent *int64
}

// reportProperties returns a map of report properties.
func (r *CableReport) reportProperties() map[string]string {
	if r.CableCurrent == nil {
		return nil
	}

	return map[string]string{
		PropertyCableCurrent: strconv.Itoa(int(*r.CableCurrent)),
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

// reportProperties returns a map of report properties taking into consideration capabilities of the
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
