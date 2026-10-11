package thermostat_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/futurehomeno/cliffhanger/adapter/service/thermostat"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedthermostat "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/thermostat"
)

func TestSetpointFromStringMap_RejectsNonFiniteTemperature(t *testing.T) {
	t.Parallel()

	for _, temp := range []string{"NaN", "+Inf", "-Inf"} {
		_, err := thermostat.SetpointFromStringMap(map[string]string{"type": "heat", "temp": temp, "unit": "C"})
		assert.Error(t, err, temp)
	}
}

func TestSetpoint_JSONUsesWireFieldNames(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal(thermostat.NewSetpoint("heat", 21.5, "C"))
	assert.NoError(t, err)
	assert.JSONEq(t, `{"type":"heat","temp":21.5,"unit":"C"}`, string(raw))
}

func TestService_SetpointReportCacheIsKeyedByNormalizedMode(t *testing.T) {
	t.Parallel()

	publisher := mockedadapter.NewServicePublisher(t)
	publisher.On("PublishServiceMessage", mock.Anything, mock.Anything).Return(nil).Once()

	svc := thermostat.NewService(publisher, &thermostat.Config{
		Specification: thermostat.Specification("test_adapter", "1", "2", nil, []string{"heat"}, []string{"heat"}, []string{thermostat.StateHeat}),
		Controller:    mockedthermostat.NewController(t).MockThermostatSetpointReport("heat", 21, "C", nil, false),
	})

	sent, err := svc.SendSetpointReport("HEAT", true)
	assert.NoError(t, err)
	assert.True(t, sent)

	sent, err = svc.SendSetpointReport("heat", false)
	assert.NoError(t, err)
	assert.False(t, sent, "an unchanged setpoint must not be re-reported under another case of its mode")
}
