package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/types"
	"github.com/futurehomeno/cliffhanger/utils"
)

func TestPhaseMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		grid   types.GridType
		phases []types.Phase
		want   types.PhaseMode
	}{
		{types.GridTypeTN, []types.Phase{types.PhaseL3, types.PhaseL1, types.PhaseL2}, types.PhaseModeNL1L2L3},
		{types.GridTypeTN, []types.Phase{types.PhaseL2, types.PhaseL1}, types.PhaseModeNL1L2},
		{types.GridTypeTN, []types.Phase{types.PhaseL2, types.PhaseL3}, types.PhaseModeNL2L3},
		{types.GridTypeTN, []types.Phase{types.PhaseL1, types.PhaseL3}, types.PhaseModeUnknown},
		{types.GridTypeTN, []types.Phase{types.PhaseL1}, types.PhaseModeNL1},
		{types.GridTypeTN, []types.Phase{types.PhaseL2}, types.PhaseModeNL2},
		{types.GridTypeTN, []types.Phase{types.PhaseL3}, types.PhaseModeNL3},
		{types.GridTypeTN, []types.Phase{types.PhaseL1, types.PhaseL1, types.PhaseL2}, types.PhaseModeUnknown},
		{types.GridTypeTN, nil, types.PhaseModeUnknown},
		{types.GridTypeIT, []types.Phase{types.PhaseL1, types.PhaseL2, types.PhaseL3}, types.PhaseModeL1L2L3},
		{types.GridTypeTT, []types.Phase{types.PhaseL1, types.PhaseL2}, types.PhaseModeL1L2},
		{types.GridTypeIT, []types.Phase{types.PhaseL3, types.PhaseL2}, types.PhaseModeL2L3},
		{types.GridTypeTT, []types.Phase{types.PhaseL1, types.PhaseL3}, types.PhaseModeL3L1},
		{types.GridTypeIT, []types.Phase{types.PhaseL1}, types.PhaseModeUnknown},
		{types.GridTypeUnknown, []types.Phase{types.PhaseL1, types.PhaseL2, types.PhaseL3}, types.PhaseModeUnknown},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, utils.PhaseMode(tt.grid, tt.phases...), "%s %v", tt.grid, tt.phases)
	}
}

func TestSupportedPhaseModes(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		[]types.PhaseMode{types.PhaseModeNL1, types.PhaseModeNL2, types.PhaseModeNL3, types.PhaseModeNL1L2L3},
		utils.SupportedPhaseModes(types.GridTypeTN, 3, types.PhaseModeNL1L2L3))
	assert.Equal(t,
		[]types.PhaseMode{types.PhaseModeL1L2, types.PhaseModeL2L3, types.PhaseModeL3L1, types.PhaseModeL1L2L3},
		utils.SupportedPhaseModes(types.GridTypeIT, 3, types.PhaseModeL1L2L3))
	assert.Equal(t, []types.PhaseMode{types.PhaseModeNL2}, utils.SupportedPhaseModes(types.GridTypeTN, 1, types.PhaseModeNL2))
	assert.Equal(t, []types.PhaseMode{types.PhaseModeL3L1}, utils.SupportedPhaseModes(types.GridTypeTT, 2, types.PhaseModeL3L1))
	assert.Equal(t, []types.PhaseMode{types.PhaseModeUnknown}, utils.SupportedPhaseModes(types.GridTypeUnknown, 3, types.PhaseModeL1L2L3))
}
