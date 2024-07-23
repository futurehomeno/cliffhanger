package parameters_test

import (
	"fmt"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
	"github.com/futurehomeno/cliffhanger/discovery"
	"github.com/futurehomeno/cliffhanger/event"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/root"
	"github.com/futurehomeno/cliffhanger/router"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedchargepoint "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/chargepoint"
	mockedparameters "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/parameters"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

const (
	testServiceName = "test_adapter"
)

func TestParametersReport(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Supported parameters report after inclusion report",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup:    testSetup(true),
				Nodes: []*suite.Node{
					{
						Name:    "Thing with parameters service",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:ad/rn:test_adapter/ad:1", "cmd.thing.get_inclusion_report", testServiceName, "1"),
						Expectations: []*suite.Expectation{
							suite.ExpectMessage("pt:j1/mt:evt/rt:ad/rn:test_adapter/ad:1", "evt.thing.inclusion_report", testServiceName),
							suite.ExpectMessage("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:1", "evt.sup_params.report", parameters.Parameters),
						},
					},
				},
			},
			{
				Name:     "No supported parameters report after inclusion report",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup:    testSetup(false),
				Nodes: []*suite.Node{
					{
						Name:    "Thing without parameters service",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:ad/rn:test_adapter/ad:1", "cmd.thing.get_inclusion_report", testServiceName, "1"),
						Expectations: []*suite.Expectation{
							suite.ExpectMessage("pt:j1/mt:evt/rt:ad/rn:test_adapter/ad:1", "evt.thing.inclusion_report", testServiceName),
							suite.ExpectMessage("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:1", "evt.sup_params.report", parameters.Parameters).Never(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func testSetup(wantParametersService bool) suite.ServiceSetup {
	return func(t *testing.T) (service suite.Service, mocks []suite.Mock) {
		t.Helper()

		mqtt := fimpgo.NewMqttTransport("tcp://localhost:11883", "parameters_test", "", "", true, 1, 1)
		eventManager := event.NewManager()
		factory := thingFactory(t, wantParametersService)

		seed := &adapter.ThingSeed{ID: "1"}

		ad := prepareAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, eventManager)
		adapterhelper.SeedAdapter(t, ad, adapter.ThingSeeds{seed})

		listener := event.NewListener(eventManager, parameters.NewInclusionReportSentEventHandler(ad))

		app, err := build(mqtt, listener, ad)
		if err != nil {
			t.Fatal("failed to build edge application:", err)
		}

		return app, nil
	}
}

func prepareAdapter(
	t *testing.T,
	workDir string,
	mqtt *fimpgo.MqttTransport,
	factory adapterhelper.FactoryHelper,
	eventManager event.Manager,
) adapter.Adapter {
	t.Helper()

	state, err := adapter.NewState(workDir)
	if err != nil {
		t.Fatal(fmt.Errorf("adapter helper: failed to create adapter state: %w", err))
	}

	a := adapter.NewAdapter(mqtt, eventManager, factory, state, "test_adapter", "1")

	return a
}

func build(mqtt *fimpgo.MqttTransport, listener event.Listener, ad adapter.Adapter) (root.App, error) {
	return root.NewEdgeAppBuilder().
		WithMQTT(mqtt).
		WithServiceDiscovery(&discovery.Resource{}).
		WithLifecycle(lifecycle.New()).
		WithTopicSubscription(
			router.TopicPatternAdapter("test_adapter"),
			router.TopicPatternDevices("test_adapter"),
		).
		WithRouting(adapter.RouteAdapter(ad)...).
		WithServices(listener).
		Build()
}

func thingFactory(t *testing.T, wantParametersService bool) adapterhelper.FactoryHelper {
	t.Helper()

	parametersCfg := &parameters.Config{
		Specification: parameters.Specification(
			"test_adapter",
			"1",
			"1",
			nil,
		),
		Controller: mockedparameters.NewController(t).MockGetParameterSpecifications(testSpecifications(t), nil, true),
	}

	chargepointCfg := &chargepoint.Config{
		Specification: chargepoint.Specification(
			"test_adapter",
			"1",
			"1",
			nil,
			[]chargepoint.State{"ready_to_charge", "charging", "error"},
			chargepoint.WithChargingModes([]string{"normal", "slow"}...),
		),
		Controller: mockedchargepoint.NewController(t),
	}

	thingCfg := &adapter.ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "1",
		},
		Connector: mockedadapter.NewDefaultConnector(t),
	}

	return func(a adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		var services []adapter.Service

		if wantParametersService {
			services = append(services, parameters.NewService(publisher, parametersCfg))
		}

		services = append(services, chargepoint.NewService(publisher, chargepointCfg))

		return adapter.NewThing(publisher, thingState, thingCfg, services...), nil
	}
}
