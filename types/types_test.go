package types_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/types"
)

func TestPhaseMode_Phases(t *testing.T) {
	t.Parallel()

	all := []types.Phase{types.PhaseL1, types.PhaseL2, types.PhaseL3}

	tests := []struct {
		mode      types.PhaseMode
		phases    []types.Phase
		effective int
	}{
		{types.PhaseModeNL1L2L3, all, 3},
		{types.PhaseModeNL1L2, []types.Phase{types.PhaseL1, types.PhaseL2}, 2},
		{types.PhaseModeNL2L3, []types.Phase{types.PhaseL2, types.PhaseL3}, 2},
		{types.PhaseModeNL1, []types.Phase{types.PhaseL1}, 1},
		{types.PhaseModeNL2, []types.Phase{types.PhaseL2}, 1},
		{types.PhaseModeNL3, []types.Phase{types.PhaseL3}, 1},
		{types.PhaseModeL1L2L3, all, 3},
		{types.PhaseModeL1L2, []types.Phase{types.PhaseL1, types.PhaseL2}, 1},
		{types.PhaseModeL2L3, []types.Phase{types.PhaseL2, types.PhaseL3}, 1},
		{types.PhaseModeL3L1, []types.Phase{types.PhaseL1, types.PhaseL3}, 1},
		{types.PhaseModeUnknown, all, 3},
		{types.PhaseMode("bogus"), all, 3},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.phases, tt.mode.Phases(), tt.mode.Str())
		assert.Equal(t, tt.effective, tt.mode.EffectivePhasesCnt(), tt.mode.Str())
	}
}
