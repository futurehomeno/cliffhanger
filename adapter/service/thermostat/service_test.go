package thermostat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter/service/thermostat"
)

func TestSetpointFromStringMap_RejectsNonFiniteTemperature(t *testing.T) {
	t.Parallel()

	for _, temp := range []string{"NaN", "+Inf", "-Inf"} {
		_, err := thermostat.SetpointFromStringMap(map[string]string{"type": "heat", "temp": temp, "unit": "C"})
		assert.Error(t, err, temp)
	}
}
