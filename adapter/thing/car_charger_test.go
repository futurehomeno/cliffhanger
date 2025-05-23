package thing_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedchargepoint "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/chargepoint"
	mockednumericmeter "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/numericmeter"
	mockedparameters "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/parameters"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestRouteCarCharger(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "other successful routing",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeCarCharger(
					mockedchargepoint.NewMockedChargepoint(
						mockedchargepoint.NewController(t).
							MockChargepointStateReport("charging", nil, false).
							MockChargepointCurrentSessionReport(&chargepoint.SessionReport{SessionEnergy: 1.74}, nil, false),
						nil,
						nil,
						nil,
						mockedchargepoint.NewAdjustableCableLockController(t).
							MockSetChargepointCableLock(true, nil, false).
							MockChargepointCableLockReport(&chargepoint.CableReport{CableLock: true}, nil, false),
					),
					mockednumericmeter.NewReporter(t).
						MockMeterReport("W", 2, nil, false).
						MockMeterReport("kWh", 123.45, nil, false),
					mockedparameters.NewController(t).
						MockGetParameterSpecifications([]*parameters.ParameterSpecification{
							{
								ID:          "cable_always_locked",
								Name:        "Cable always locked",
								Description: "Tells if the cable is always locked",
								ValueType:   "bool",
								Options: parameters.SelectOptions{
									{Label: "Yes", Value: true},
									{Label: "No", Value: false},
								},
								DefaultValue: false,
								ReadOnly:     true,
							},
						}, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "set cable lock",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.cable_lock.set", "chargepoint", true),
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.cable_lock.report", "chargepoint", true),
						},
					},
					{
						Name:    "cable lock report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.cable_lock.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.cable_lock.report", "chargepoint", true),
						},
					},
					{
						Name:    "state report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.state.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.state.report", "chargepoint", "charging"),
						},
					},
					{
						Name:    "current session report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.current_session.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.current_session.report", "chargepoint", 1.74),
						},
					},
					{
						Name:    "power",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec", "W"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 2).
								ExpectProperty("unit", "W"),
						},
					},
					{
						Name:    "energy",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec", "kWh"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 123.45).
								ExpectProperty("unit", "kWh"),
						},
					},
					{
						Name:    "parameters report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:2", "cmd.sup_params.get_report", "parameters"),
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:2", "evt.sup_params.report", "parameters", []parameters.ParameterSpecification{
								{
									ID:          "cable_always_locked",
									Name:        "Cable always locked",
									Description: "Tells if the cable is always locked",
									ValueType:   "bool",
									Options: parameters.SelectOptions{
										{Label: "Yes", Value: true},
										{Label: "No", Value: false},
									},
									DefaultValue: false,
									ReadOnly:     true,
								},
							}),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestTaskCarCharger(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Car charger tasks",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskCarCharger(
					mockedchargepoint.NewMockedChargepoint(
						mockedchargepoint.NewController(t).
							MockChargepointCurrentSessionReport(&chargepoint.SessionReport{SessionEnergy: 1.23}, nil, false).
							MockChargepointStateReport("ready_to_charge", nil, false),
						nil,
						nil,
						nil,
						mockedchargepoint.NewAdjustableCableLockController(t).
							MockChargepointCableLockReport(&chargepoint.CableReport{CableLock: true}, nil, false),
					),
					mockednumericmeter.NewReporter(t).
						MockMeterReport("W", 1500, nil, false).
						MockMeterReport("kWh", 123.56, nil, false),
					100*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "Successful reporting",
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.cable_lock.report", "chargepoint", true),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.current_session.report", "chargepoint", 1.23),
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.state.report", "chargepoint", "ready_to_charge").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 1500).ExpectProperty("unit", "W"),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 123.56).ExpectProperty("unit", "kWh"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeCarCharger(
	chargepointController chargepoint.Controller,
	meterElecReporter *mockednumericmeter.Reporter,
	specificationReporter *mockedparameters.Controller,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupCarCharger(t, mqtt, chargepointController, meterElecReporter, specificationReporter, 0)

		return routing, nil, mocks
	}
}

func taskCarCharger(
	chargepointController chargepoint.Controller,
	meterElecReporter *mockednumericmeter.Reporter,
	interval time.Duration,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		_, tasks, mocks := setupCarCharger(t, mqtt, chargepointController, meterElecReporter, nil, interval)

		return nil, tasks, mocks
	}
}

func setupCarCharger(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	chargepointController chargepoint.Controller,
	meterElecReporter *mockednumericmeter.Reporter,
	specificationController *mockedparameters.Controller,
	duration time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mockedController, ok := chargepointController.(suite.Mock)
	if !ok {
		t.Fatal("controller is not a mock")
	}

	mocks := []suite.Mock{mockedController}

	cfg := &thing.CarChargerConfig{
		ThingConfig: &adapter.ThingConfig{
			InclusionReport: &fimptype.ThingInclusionReport{
				Address: "2",
			},
			Connector: mockedadapter.NewDefaultConnector(t),
		},
		ChargepointConfig: &chargepoint.Config{
			Specification: chargepoint.Specification(
				"test_adapter",
				"1",
				"2",
				nil,
				[]chargepoint.State{"ready_to_charge", "charging", "error"},
			),
			Controller: chargepointController,
		},
	}

	if meterElecReporter != nil {
		cfg.MeterElecConfig = &numericmeter.Config{
			Specification: numericmeter.Specification(
				numericmeter.MeterElec,
				"test_adapter",
				"1",
				"2",
				nil,
				[]numericmeter.Unit{"W", "kWh"},
			),
			Reporter: meterElecReporter,
		}

		mocks = append(mocks, meterElecReporter)
	}

	if specificationController != nil {
		cfg.ParameterServiceConfig = &parameters.Config{
			Specification: parameters.Specification(
				"test_adapter",
				"1",
				"2",
				nil,
			),
			Controller: specificationController,
		}

		mocks = append(mocks, specificationController)
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(adapter adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return thing.NewCarCharger(publisher, thingState, cfg), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return thing.RouteCarCharger(ad), thing.TaskCarCharger(ad, duration), mocks
}
