package waterheater_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter/service/waterheater"
	mockedwaterheater "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/waterheater"
)

func TestService_SetSetpointUsesPerModeRange(t *testing.T) {
	t.Parallel()

	controller := mockedwaterheater.NewController(t).
		MockSetWaterHeaterSetpoint("heat", 70, "C", nil, true).
		MockSetWaterHeaterSetpoint("eco", 52.5, "C", nil, true)

	svc := waterheater.NewService(nil, &waterheater.Config{
		Specification: waterheater.Specification(
			"test_adapter", "1", "2", nil, testModes, testSetpoints, testStates,
			&waterheater.Range{Min: 30, Max: 80},
			map[string]waterheater.Range{"eco": {Min: 40, Max: 60}},
			testStep,
		),
		Controller: controller,
	})

	assert.ErrorContains(t, svc.SetSetpoint("ECO", 70, "C"), "out of range")
	assert.NoError(t, svc.SetSetpoint("heat", 70, "C"), "a mode without its own range falls back to sup_range")
	assert.NoError(t, svc.SetSetpoint("eco", 52.3, "C"), "the value is rounded to sup_step before the range check")
}

func TestService_SetSetpointWithoutRangeOrStep(t *testing.T) {
	t.Parallel()

	controller := mockedwaterheater.NewController(t).
		MockSetWaterHeaterSetpoint("heat", 123.4, "C", nil, true)

	svc := waterheater.NewService(nil, &waterheater.Config{
		Specification: waterheater.Specification("test_adapter", "1", "2", nil, testModes, testSetpoints, testStates, nil, nil, 0),
		Controller:    controller,
	})

	assert.NoError(t, svc.SetSetpoint("heat", 123.4, "C"))
}
