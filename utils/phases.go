package utils

import (
	"slices"

	"github.com/futurehomeno/cliffhanger/types"
)

func Phases() []types.Phase {
	return []types.Phase{types.PhaseL1, types.PhaseL2, types.PhaseL3}
}

func PhaseModes() []types.PhaseMode {
	return []types.PhaseMode{
		types.PhaseModeNL1L2L3,
		types.PhaseModeNL1L2,
		types.PhaseModeNL2L3,
		types.PhaseModeNL1,
		types.PhaseModeNL2,
		types.PhaseModeNL3,
		types.PhaseModeL1L2L3,
		types.PhaseModeL1L2,
		types.PhaseModeL2L3,
		types.PhaseModeL3L1,
	}
}

func GridTypes() []types.GridType {
	return []types.GridType{
		types.GridTypeTN,
		types.GridTypeTT,
		types.GridTypeIT,
	}
}

func PhaseMode(gridType types.GridType, phases ...types.Phase) types.PhaseMode { //nolint:cyclop
	switch gridType {
	case types.GridTypeTN:
		switch len(phases) {
		case 3:
			if slices.Contains(phases, types.PhaseL1) && slices.Contains(phases, types.PhaseL2) && slices.Contains(phases, types.PhaseL3) {
				return types.PhaseModeNL1L2L3
			}
		case 2:
			if slices.Contains(phases, types.PhaseL1) && slices.Contains(phases, types.PhaseL2) {
				return types.PhaseModeNL1L2
			} else if slices.Contains(phases, types.PhaseL2) && slices.Contains(phases, types.PhaseL3) {
				return types.PhaseModeNL2L3
			}
		case 1:
			switch phases[0] {
			case types.PhaseL1:
				return types.PhaseModeNL1

			case types.PhaseL2:
				return types.PhaseModeNL2

			case types.PhaseL3:
				return types.PhaseModeNL3
			}
		}

	case types.GridTypeIT, types.GridTypeTT:
		switch len(phases) {
		case 3:
			if slices.Contains(phases, types.PhaseL1) && slices.Contains(phases, types.PhaseL2) && slices.Contains(phases, types.PhaseL3) {
				return types.PhaseModeL1L2L3
			}
		case 2:
			if slices.Contains(phases, types.PhaseL1) && slices.Contains(phases, types.PhaseL2) { //nolint:gocritic
				return types.PhaseModeL1L2
			} else if slices.Contains(phases, types.PhaseL2) && slices.Contains(phases, types.PhaseL3) {
				return types.PhaseModeL2L3
			} else if slices.Contains(phases, types.PhaseL3) && slices.Contains(phases, types.PhaseL1) {
				return types.PhaseModeL3L1
			}
		}
	}

	return types.PhaseModeUnknown
}

func SupportedPhaseModes(earthingType types.GridType, utilizedPhases int, phaseMode types.PhaseMode) []types.PhaseMode {
	switch earthingType {
	case types.GridTypeTN:
		if utilizedPhases == 3 {
			return []types.PhaseMode{types.PhaseModeNL1, types.PhaseModeNL2, types.PhaseModeNL3, types.PhaseModeNL1L2L3}
		}
		return []types.PhaseMode{phaseMode}

	case types.GridTypeIT, types.GridTypeTT:
		if utilizedPhases == 3 {
			return []types.PhaseMode{types.PhaseModeL1L2, types.PhaseModeL2L3, types.PhaseModeL3L1, types.PhaseModeL1L2L3}
		}
		return []types.PhaseMode{phaseMode}

	case types.GridTypeUnknown:
	}

	return []types.PhaseMode{types.PhaseModeUnknown}
}
