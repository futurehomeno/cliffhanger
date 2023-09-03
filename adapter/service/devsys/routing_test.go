package devsys_test

import (
	"errors"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/devsys"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockeddevsys "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/devsys"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestRouteCarCharger(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "successful reboot",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockeddevsys.NewRebootController(t).
						MockRebootDevice(false, nil, false).
						MockRebootDevice(true, nil, false),
				),
				Nodes: []*suite.Node{
					{
						Name:    "reboot with unspecified mode",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:dev_sys/ad:2", "cmd.thing.reboot", "dev_sys"),
						Expectations: []*suite.Expectation{
							suite.ExpectSuccess("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:dev_sys/ad:2", "dev_sys"),
						},
					},
					{
						Name:    "reboot with specified mode",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:dev_sys/ad:2", "cmd.thing.reboot", "dev_sys", true),
						Expectations: []*suite.Expectation{
							suite.ExpectSuccess("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:dev_sys/ad:2", "dev_sys"),
						},
					},
				},
			},
			{
				Name:     "reboot is unsupported",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockeddevsys.NewController(t),
				),
				Nodes: []*suite.Node{
					{
						Name:    "reboot is unsupported",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:dev_sys/ad:2", "cmd.thing.reboot", "dev_sys"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:dev_sys/ad:2", "dev_sys"),
						},
					},
				},
			},
			{
				Name:     "other failed routing",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockeddevsys.NewRebootController(t).
						MockRebootDevice(false, errors.New("test"), false),
				),
				Nodes: []*suite.Node{
					{
						Name:    "reboot failed",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:dev_sys/ad:2", "cmd.thing.reboot", "dev_sys"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:dev_sys/ad:2", "dev_sys"),
						},
					},
					{
						Name:    "reboot failed - service not found",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:dev_sys/ad:3", "cmd.thing.reboot", "dev_sys"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:dev_sys/ad:3", "dev_sys"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeService(
	controller devsys.Controller,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupService(t, mqtt, controller)

		return routing, nil, mocks
	}
}

func setupService(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	controller devsys.Controller,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mockedController, ok := controller.(suite.Mock)
	if !ok {
		t.Fatal("controller is not a mock")
	}

	mocks := []suite.Mock{mockedController}

	thingCfg := &adapter.ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "2",
		},
		Connector: mockedadapter.NewConnector(t),
	}

	devSysCfg := &devsys.Config{
		Specification: devsys.Specification(
			"test_adapter",
			"1",
			"2",
			nil,
		),
		Controller: controller,
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(a adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return adapter.NewThing(publisher, thingState, thingCfg, devsys.NewService(publisher, devSysCfg)), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return devsys.RouteService(ad), nil, mocks
}
