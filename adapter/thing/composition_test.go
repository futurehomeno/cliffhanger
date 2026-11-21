package thing_test

import (
	"testing"

	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/alarm"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/devsys"
	"github.com/futurehomeno/cliffhanger/adapter/service/diagnostic"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericsensor"
	"github.com/futurehomeno/cliffhanger/adapter/service/ota"
	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
	"github.com/futurehomeno/cliffhanger/adapter/service/presence"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedchargepoint "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/chargepoint"
	mockednumericmeter "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/numericmeter"
	mockednumericsensor "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/numericsensor"
	mockedota "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/ota"
	mockedparameters "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/parameters"
	mockedpresence "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/presence"
)

func serviceNames(th adapter.Thing) []fimptype.ServiceNameT {
	var names []fimptype.ServiceNameT
	for _, s := range th.InclusionReport().Services {
		names = append(names, s.Name)
	}

	return names
}

func TestNewCarCharger_AllOptionalServices(t *testing.T) {
	t.Parallel()

	stale := fimptype.Service{Name: "out_bin_switch", Address: "/rt:dev/rn:test_adapter/ad:1/sv:out_bin_switch/ad:2"}

	th := thing.NewCarCharger(nil, mockedadapter.NewThingState(t), &thing.CarChargerConfig{
		ThingConfig: &adapter.ThingConfig{
			InclusionReport: &fimptype.ThingInclusionReport{Address: "2", Services: []fimptype.Service{stale}},
		},
		ChargepointConfig: &chargepoint.Config{
			Specification: chargepoint.Specification("test_adapter", "1", "2", nil, nil),
			Controller:    mockedchargepoint.NewController(t),
		},
		AlarmConfig:            &alarm.Config{Specification: alarm.Specification("test_adapter", "1", "2", nil, nil)},
		DevSysConfig:           &devsys.Config{Specification: devsys.Specification("test_adapter", "1", "2", nil)},
		DiagnosticConfig:       &diagnostic.Config{Specification: diagnostic.Specification("test_adapter", "1", "2", nil)},
		MeterElecConfig:        &numericmeter.Config{Specification: numericmeter.Specification(numericmeter.MeterElec, "test_adapter", "1", "2", nil, numericmeter.Units{numericmeter.UnitW}), Reporter: mockednumericmeter.NewReporter(t)},
		OTAConfig:              &ota.Config{Specification: ota.Specification("test_adapter", "1", "2", nil), Controller: mockedota.NewController(t)},
		ParameterServiceConfig: &parameters.Config{Specification: parameters.Specification("test_adapter", "1", "2", nil), Controller: mockedparameters.NewController(t)},
	})

	assert.ElementsMatch(t, []fimptype.ServiceNameT{
		chargepoint.Chargepoint, alarm.AlarmSystem, devsys.DevSys, diagnostic.Diagnostic,
		numericmeter.MeterElec, ota.OTA, parameters.Parameters,
	}, serviceNames(th), "the inclusion report must list exactly the constructed services")
	assert.Len(t, th.Services(""), 7)
}

func TestNewSensor_PresenceAndNumericSensors(t *testing.T) {
	t.Parallel()

	sensorCfg := func(name fimptype.ServiceNameT) *numericsensor.Config {
		return &numericsensor.Config{
			Specification: numericsensor.Specification("test_adapter", "1", name, "2", nil, []string{"C"}),
			Reporter:      mockednumericsensor.NewReporter(t),
		}
	}

	th := thing.NewSensor(nil, mockedadapter.NewThingState(t), &thing.SensorConfig{
		ThingConfig:          &adapter.ThingConfig{InclusionReport: &fimptype.ThingInclusionReport{Address: "2"}},
		NumericSensorConfigs: []*numericsensor.Config{sensorCfg(numericsensor.SensorTemp), sensorCfg(numericsensor.SensorHumid)},
		PresenceConfig: &presence.Config{
			Specification: presence.Specification("test_adapter", "1", "2", nil),
			Controller:    mockedpresence.NewController(t),
		},
	})

	assert.Equal(t, []fimptype.ServiceNameT{presence.SensorPresence, numericsensor.SensorTemp, numericsensor.SensorHumid}, serviceNames(th))
}
