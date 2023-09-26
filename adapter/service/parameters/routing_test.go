package parameters_test

import (
	"errors"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedparameters "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/parameters"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

var ( //nolint:gofumpt
	errTest = errors.New("test error")
)

func TestRouteService(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "successful parameters reporting",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedparameters.NewController(t).
						MockGetParameterSpecifications(testSpecifications(t), nil, false).
						MockGetParameter("1", testParameter(t), nil, false).
						MockSetParameter(testParameter(t), nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "get parameter specifications",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:2", "cmd.sup_params.get_report", "parameters"),
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:2", "evt.sup_params.report", "parameters", testSpecifications(t)),
						},
					},
					{
						Name:    "get parameter",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:2", "cmd.param.get_report", "parameters", "1"),
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:2", "evt.param.report", "parameters", testParameter(t)),
						},
					},
					{
						Name:    "set parameter",
						Command: suite.ObjectMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:2", "cmd.param.set", "parameters", testParameter(t)),
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:2", "evt.param.report", "parameters", testParameter(t)),
						},
					},
				},
			},
			{
				Name:     "failed parameters reporting due to controller errors",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedparameters.NewController(t).
						MockGetParameterSpecifications(testSpecifications(t), errTest, false).
						MockGetParameter("1", testParameter(t), errTest, false),
				),
				Nodes: []*suite.Node{
					{
						Name:    "get parameter specifications",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:2", "cmd.sup_params.get_report", "parameters"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:2", "parameters"),
						},
					},
					{
						Name:    "get parameter",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:2", "cmd.param.get_report", "parameters", "1"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:2", "parameters"),
						},
					},
					{
						Name:    "set parameter - error when getting param specifications",
						Command: suite.ObjectMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:2", "cmd.param.set", "parameters", testParameter(t)),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:2", "parameters"),
						},
					},
				},
			},
			{
				Name:     "error when setting a parameter - controller's setter error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedparameters.NewController(t).
						MockGetParameterSpecifications(testSpecifications(t), nil, false).
						MockSetParameter(testParameter(t), errTest, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "set parameter",
						Command: suite.ObjectMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:2", "cmd.param.set", "parameters", testParameter(t)),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:2", "parameters"),
						},
					},
				},
			},
			{
				Name:     "error when setting a parameter - read-only parameter",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedparameters.NewController(t).
						MockGetParameterSpecifications(testSpecifications(t), nil, false),
				),
				Nodes: []*suite.Node{
					{
						Name:    "set parameter",
						Command: suite.ObjectMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:2", "cmd.param.set", "parameters", testReadOnlyParameter(t)),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:parameters/ad:2", "parameters"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeService(
	controller parameters.Controller,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
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

		serviceCfg := &parameters.Config{
			Specification: parameters.Specification(
				"test_adapter",
				"1",
				"2",
				nil,
			),
			Controller: controller,
		}

		seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

		factory := adapterhelper.FactoryHelper(func(a adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
			return adapter.NewThing(publisher, thingState, thingCfg, parameters.NewService(publisher, serviceCfg)), nil
		})

		ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

		return parameters.RouteService(ad), nil, mocks
	}
}

func testSpecifications(t *testing.T) []*parameters.ParameterSpecification {
	t.Helper()

	return []*parameters.ParameterSpecification{
		{
			ID:          "1",
			Name:        "Example select parameter",
			Description: "Example long description of the parameter.",
			ValueType:   parameters.ValueTypeInt,
			WidgetType:  parameters.WidgetTypeSelect,
			Options: parameters.SelectOptions{
				{
					Label: "Example option 1",
					Value: 0,
				},
				{
					Label: "Example option 2",
					Value: 1,
				},
				{
					Label: "Example option 3",
				},
			},
			DefaultValue: 0,
			ReadOnly:     false,
		},
		(&parameters.ParameterSpecification{
			ID:           "2",
			Name:         "Example input parameter",
			Description:  "Example long description of the parameter.",
			ValueType:    parameters.ValueTypeInt,
			WidgetType:   parameters.WidgetTypeInput,
			DefaultValue: 0,
			ReadOnly:     true,
		}).WithMin(-5).WithMax(5),
	}
}

func testParameter(t *testing.T) *parameters.Parameter {
	t.Helper()

	return parameters.NewIntParameter("1", 1)
}

func testReadOnlyParameter(t *testing.T) *parameters.Parameter {
	t.Helper()

	return parameters.NewIntParameter("2", 2)
}
