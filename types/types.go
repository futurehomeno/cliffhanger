package types

const (
	PhaseL1 Phase = "L1"
	PhaseL2 Phase = "L2"
	PhaseL3 Phase = "L3"

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

type Phase string

func (p Phase) Str() string {
	return string(p)
}

type GridType string

func (t GridType) Str() string {
	return string(t)
}

type PhaseMode string

func (m PhaseMode) Str() string {
	return string(m)
}

func (m PhaseMode) Phases() []Phase {
	switch m {
	case PhaseModeNL1L2L3:
		return []Phase{PhaseL1, PhaseL2, PhaseL3}
	case PhaseModeNL1:
		return []Phase{PhaseL1}
	case PhaseModeNL2:
		return []Phase{PhaseL2}
	case PhaseModeNL3:
		return []Phase{PhaseL3}
	case PhaseModeL1L2L3:
		return []Phase{PhaseL1, PhaseL2, PhaseL3}
	case PhaseModeL1L2:
		return []Phase{PhaseL1, PhaseL2}
	case PhaseModeL2L3:
		return []Phase{PhaseL2, PhaseL3}
	case PhaseModeL3L1:
		return []Phase{PhaseL1, PhaseL3}
	default:
		return []Phase{PhaseL1, PhaseL2, PhaseL3}
	}
}
