package outlvlswitch_test

import (
	"testing"

	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/router"
	mockedoutlvlswitch "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/outlvlswitch"
)

// TestService_SetLevelRejectsAnInvertedRange pins that a spec whose min_lvl is above its max_lvl
// fails loudly. Clamping into an inverted range collapses every level to max_lvl, so a caller that
// swapped Specification's maxLvl and minLvl arguments would otherwise send one fixed level to the
// device for every command without a word.
func TestService_SetLevelRejectsAnInvertedRange(t *testing.T) {
	t.Parallel()

	s := outlvlswitch.NewService(nil, &outlvlswitch.Config{
		Specification: &fimptype.Service{
			Name:    outlvlswitch.OutLvlSwitch,
			Address: "test",
			Props: map[string]any{
				outlvlswitch.PropertyMinLvl: 254,
				outlvlswitch.PropertyMaxLvl: 1,
			},
		},
	})

	assert.ErrorContains(t, s.SetLevel(50, nil), "min_lvl 254 is above max_lvl 1")
}

func TestService_LevelTransitionWithoutControllerIsNotSupported(t *testing.T) {
	t.Parallel()

	s := outlvlswitch.NewService(nil, &outlvlswitch.Config{
		Specification: outlvlswitch.Specification("test_adapter", "1", "2", "", 99, 0, nil),
		Controller:    mockedoutlvlswitch.NewController(t),
	})

	for _, err := range []error{s.StartLevelTransition(outlvlswitch.TransitionUp, outlvlswitch.LevelTransitionParams{}), s.StopLevelTransition()} {
		var coded *router.Error

		require.ErrorAs(t, err, &coded)
		assert.Equal(t, fimptype.ErrorCodeNotSupported, coded.Code)
	}
}
