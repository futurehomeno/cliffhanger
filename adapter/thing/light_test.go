package thing_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/colorctrl"
	"github.com/futurehomeno/cliffhanger/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedcolorctrl "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/colorctrl"
	mockedoutlvlswitch "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestRouteLight(t *testing.T) { //nolint:paralleltest
	validColor := map[string]int64{"red": 255, "green": 55, "blue": 100}
	invalidColor := map[string]float64{"red": 255.0, "green": 55.0, "blue": 100.0}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "successful set level routing",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t).
						MockSetLevelSwitchLevel(99, time.Duration(1)*time.Second, nil, true).
						MockLevelSwitchLevelReport(99, nil, true).
						MockSetLevelSwitchLevel(98, time.Duration(0), nil, true).
						MockLevelSwitchLevelReport(98, nil, true),
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name: "set level with duration",
						Commands: []*fimpgo.Message{suite.NewMessageBuilder().
							IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.set", "out_lvl_switch", 99).
							AddProperty("duration", "1").
							Build()},
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "evt.lvl.report", "out_lvl_switch", 99),
						},
					},
					{
						Name:     "set level without duration",
						Commands: []*fimpgo.Message{suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.set", "out_lvl_switch", 98)},
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "evt.lvl.report", "out_lvl_switch", 98),
						},
					},
				},
			},
			{
				Name:     "successful get level report",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t).
						MockLevelSwitchLevelReport(99, nil, true),
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name: "get report",
						Commands: []*fimpgo.Message{suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.get_report", "out_lvl_switch").
							Build()},
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "evt.lvl.report", "out_lvl_switch", 99),
						},
					},
				},
			},
			{
				Name:     "failed set level - setting error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t).
						MockSetLevelSwitchLevel(99, time.Duration(0), errors.New("setting error"), true),
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name:     "set level",
						Commands: []*fimpgo.Message{suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.set", "out_lvl_switch", 99)},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
					{
						Name:     "wrong value type",
						Commands: []*fimpgo.Message{suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.set", "out_lvl_switch", "99")},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
					{
						Name:     "wrong address",
						Commands: []*fimpgo.Message{suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:3", "cmd.lvl.set", "out_lvl_switch", 99)},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:3", "out_lvl_switch"),
						},
					},
					{
						Name: "wrong address and wrong format of duration",
						Commands: []*fimpgo.Message{suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:3", "cmd.lvl.get_report", "out_lvl_switch").
							Build()},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:3", "out_lvl_switch"),
						},
					},
					{
						Name: "set level with wrong format of duration",
						Commands: []*fimpgo.Message{suite.NewMessageBuilder().
							IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.set", "out_lvl_switch", 99).
							AddProperty("duration", "1s").
							Build()},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
				},
			},
			{
				Name:     "failed set level - level report error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t).
						MockSetLevelSwitchLevel(99, time.Duration(0), nil, true).
						MockLevelSwitchLevelReport(99, errors.New("report error"), true),
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name:     "set level",
						Commands: []*fimpgo.Message{suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.set", "out_lvl_switch", 99)},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
				},
			},
			{
				Name:     "failed set binary - setting error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t).
						MockSetLevelSwitchBinaryState(true, errors.New("setting error"), true),
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name:     "set binary",
						Commands: []*fimpgo.Message{suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.binary.set", "out_lvl_switch", true)},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
					{
						Name:     "wrong value type",
						Commands: []*fimpgo.Message{suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.binary.set", "out_lvl_switch", "true")},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
					{
						Name:     "wrong address",
						Commands: []*fimpgo.Message{suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:3", "cmd.binary.set", "out_lvl_switch", true)},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:3", "out_lvl_switch"),
						},
					},
				},
			},
			{
				Name:     "failed set binary - level report error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t).
						MockSetLevelSwitchBinaryState(true, nil, true).
						MockLevelSwitchLevelReport(99, errors.New("report error"), true),
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name:     "set binary",
						Commands: []*fimpgo.Message{suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.binary.set", "out_lvl_switch", true)},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
				},
			},
			{
				Name:     "failed get level - send level error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t).
						MockLevelSwitchLevelReport(99, errors.New("sending error"), true),
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name:     "get level",
						Commands: []*fimpgo.Message{suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "cmd.lvl.get_report", "out_lvl_switch")},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "out_lvl_switch"),
						},
					},
				},
			},
			{
				Name:     "successful set color",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t),
					mockedcolorctrl.NewController(t).
						MockSetColorCtrlColor(validColor, nil, true).
						MockColorCtrlColorReport(validColor, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:     "set color",
						Commands: []*fimpgo.Message{suite.IntMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.set", "color_ctrl", validColor)},
						Expectations: []*suite.Expectation{
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "evt.color.report", "color_ctrl", validColor),
						},
					},
				},
			},
			{
				Name:     "successful get color",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t),
					mockedcolorctrl.NewController(t).
						MockColorCtrlColorReport(validColor, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:     "get color",
						Commands: []*fimpgo.Message{suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.get_report", "color_ctrl")},
						Expectations: []*suite.Expectation{
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "evt.color.report", "color_ctrl", validColor),
						},
					},
				},
			},
			{
				Name:     "failed set color level - setting error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t),
					mockedcolorctrl.NewController(t).
						MockSetColorCtrlColor(validColor, errors.New("error"), true),
				),
				Nodes: []*suite.Node{
					{
						Name:     "controller error",
						Commands: []*fimpgo.Message{suite.IntMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.set", "color_ctrl", validColor)},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "color_ctrl"),
						},
					},
					{
						Name:     "wrong colorValue type",
						Commands: []*fimpgo.Message{suite.FloatMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.set", "color_ctrl", invalidColor)},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "color_ctrl"),
						},
					},
					{
						Name:     "wrong address",
						Commands: []*fimpgo.Message{suite.IntMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:3", "cmd.color.set", "color_ctrl", validColor)},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:3", "color_ctrl"),
						},
					},
				},
			},
			{
				Name:     "failed set color level - report error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeLight(
					mockedoutlvlswitch.NewController(t),
					mockedcolorctrl.NewController(t).
						MockSetColorCtrlColor(validColor, nil, true).
						MockColorCtrlColorReport(validColor, errors.New("error"), true),
				),
				Nodes: []*suite.Node{
					{
						Name:     "report error",
						Commands: []*fimpgo.Message{suite.IntMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "cmd.color.set", "color_ctrl", validColor)},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "color_ctrl"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestTaskLight(t *testing.T) { //nolint:paralleltest
	color1 := map[string]int64{
		"red":   255,
		"green": 55,
		"blue":  100,
	}

	color2 := map[string]int64{
		"red":   55,
		"green": 155,
		"blue":  255,
	}

	color3 := map[string]int64{
		"red":   100,
		"green": 200,
		"blue":  0,
	}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Light with level tasks",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskLight(
					mockedoutlvlswitch.NewController(t).
						MockLevelSwitchLevelReport(99, nil, true).
						MockLevelSwitchLevelReport(99, errors.New("task error"), true).
						MockLevelSwitchLevelReport(98, nil, true).
						MockLevelSwitchLevelReport(97, nil, true).
						MockLevelSwitchLevelReport(97, nil, false),
					nil,
					100*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "Two reports",
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "evt.lvl.report", "out_lvl_switch", 99),
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "evt.lvl.report", "out_lvl_switch", 98),
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:out_lvl_switch/ad:2", "evt.lvl.report", "out_lvl_switch", 97),
						},
					},
				},
			},
			{
				Name:     "Light with color tasks",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskLight(
					mockedoutlvlswitch.NewController(t).MockLevelSwitchLevelReport(100, nil, false),
					mockedcolorctrl.NewController(t).
						MockColorCtrlColorReport(color1, nil, true).
						MockColorCtrlColorReport(color1, errors.New("task error"), true).
						MockColorCtrlColorReport(color2, nil, true).
						MockColorCtrlColorReport(color3, nil, true).
						MockColorCtrlColorReport(color3, nil, false),
					100*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "Tasks",
						Expectations: []*suite.Expectation{
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "evt.color.report", "color_ctrl", color1),
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "evt.color.report", "color_ctrl", color2),
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "evt.color.report", "color_ctrl", color3),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeLight(
	lightController *mockedoutlvlswitch.Controller,
	colorCtrlController *mockedcolorctrl.Controller,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupLight(t, mqtt, lightController, colorCtrlController, 0)

		return routing, nil, mocks
	}
}

func taskLight(
	lightController *mockedoutlvlswitch.Controller,
	colorCtrlController *mockedcolorctrl.Controller,
	interval time.Duration,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		_, tasks, mocks := setupLight(t, mqtt, lightController, colorCtrlController, interval)

		return nil, tasks, mocks
	}
}

func setupLight(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	lightController *mockedoutlvlswitch.Controller,
	colorCtrlController *mockedcolorctrl.Controller,
	duration time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mocks := []suite.Mock{lightController}

	cfg := &thing.LightConfig{
		ThingConfig: &adapter.ThingConfig{
			InclusionReport: &fimptype.ThingInclusionReport{
				Address: "2",
			},
			Connector: mockedadapter.NewDefaultConnector(t),
		},
		OutLvlSwitchConfig: &outlvlswitch.Config{
			Specification: outlvlswitch.Specification(
				"test_adapter",
				"1",
				"2",
				outlvlswitch.SwitchTypeOnAndOff,
				99,
				0,
				nil,
			),
			Controller: lightController,
		},
	}

	if colorCtrlController != nil {
		cfg.ColorCtrlConfig = &colorctrl.Config{
			Specification: colorctrl.Specification(
				"test_adapter",
				"1",
				"2",
				nil,
				[]string{"red", "green", "blue"},
				map[string]int64{"min": 180, "max": 7620, "step": 60},
			),
			Controller: colorCtrlController,
		}
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(_ adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return thing.NewLight(publisher, thingState, cfg), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return thing.RouteLight(ad), thing.TaskLight(ad, duration), mocks
}
