package outbinswitch_test

import (
	"errors"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/outbinswitch"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedoutbinswitch "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/outbinswitch"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestRouteService(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Route out bin switch tests",
				TearDown: adapterhelper.TearDownAdapter("../../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedoutbinswitch.NewController(t).
						MockedBinarySwitchBinarySet(true, nil, true).
						MockedBinarySwitchBinaryReport(true, nil, true).
						MockedBinarySwitchBinaryReport(false, nil, true).
						MockedBinarySwitchBinaryReport(true, errors.New("report error"), true).
						MockedBinarySwitchBinarySet(true, errors.New("setting error"), true).
						MockedBinarySwitchBinarySet(true, nil, true).
						MockedBinarySwitchBinaryReport(true, errors.New("setting error"), true),
				),
				Nodes: []*suite.Node{
					{
						Name:     "Switch binary on",
						Commands: []*fimpgo.Message{suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_bin_switch/ad:2", "cmd.binary.set", "out_bin_switch", true)},
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_bin_switch/ad:2", "evt.binary.report", "out_bin_switch", true),
						},
					},
					{
						Name:     "Get binary report",
						Commands: []*fimpgo.Message{suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_bin_switch/ad:2", "cmd.binary.get_report", "out_bin_switch")},
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_bin_switch/ad:2", "evt.binary.report", "out_bin_switch", false),
						},
					},
					{
						Name:     "Wrong topic",
						Commands: []*fimpgo.Message{suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_bin_switch/ad:666", "cmd.binary.get_report", "out_bin_switch")},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_bin_switch/ad:666", "out_bin_switch"),
						},
					},
					{
						Name:     "Get errored binary report",
						Commands: []*fimpgo.Message{suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_bin_switch/ad:2", "cmd.binary.get_report", "out_bin_switch")},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_bin_switch/ad:2", "out_bin_switch"),
						},
					},
					{
						Name:     "Switch binary on - wrong topic",
						Commands: []*fimpgo.Message{suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_bin_switch/ad:666", "cmd.binary.set", "out_bin_switch", true)},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_bin_switch/ad:666", "out_bin_switch"),
						},
					},
					{
						Name:     "Switch binary on - wrong type",
						Commands: []*fimpgo.Message{suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_bin_switch/ad:2", "cmd.binary.set", "out_bin_switch", true)},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_bin_switch/ad:2", "out_bin_switch"),
						},
					},
					{
						Name:     "Switch binary on - report error",
						Commands: []*fimpgo.Message{suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_bin_switch/ad:2", "cmd.binary.set", "out_bin_switch", true)},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_bin_switch/ad:2", "out_bin_switch"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeService(controller outbinswitch.Controller, options ...adapter.SpecificationOption) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupService(t, mqtt, controller, options...)

		return routing, nil, mocks
	}
}

func setupService(t *testing.T, mqtt *fimpgo.MqttTransport, controller outbinswitch.Controller, options ...adapter.SpecificationOption) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mockedController, ok := controller.(suite.Mock)
	if !ok {
		t.Fatal("controller must be a mock")
	}

	mocks := []suite.Mock{mockedController}
	thingCfg := &adapter.ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "2",
		},
		Connector: mockedadapter.NewDefaultConnector(t),
	}

	switchCfg := &outbinswitch.Config{
		Specification: outbinswitch.Specification(
			"test_adapter",
			"1",
			"2",
			nil,
			options...,
		),
		Controller: controller,
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(a adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return adapter.NewThing(publisher, thingState, thingCfg, outbinswitch.NewService(publisher, switchCfg)), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return outbinswitch.RouteService(ad), task.Combine(outbinswitch.TaskReporting(ad, 0)), mocks
}
