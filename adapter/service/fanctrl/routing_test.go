package fanctrl_test

import (
	"errors"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/fanctrl"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedfanctrl "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/fanctrl"
	cliffSuite "github.com/futurehomeno/cliffhanger/test/suite"
)

func TestRouteService(t *testing.T) { //nolint:paralleltest
	s := &cliffSuite.Suite{
		Cases: []*cliffSuite.Case{
			{
				Name:     "fan ctrl get report",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(mockedfanctrl.NewController(t).
					MockGetMode("normal", nil, true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "Cmd mode get report",
						Command: cliffSuite.NewMessageBuilder().
							NullMessage(
								"pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2",
								"cmd.mode.get_report",
								"fan_ctrl",
							).
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2", "evt.mode.report", "fan_ctrl", "normal"),
						},
					},
				},
			},
			{
				Name:     "fan ctrl set report",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(mockedfanctrl.NewController(t).
					MockSetMode("night", nil, true).
					MockGetMode("night", nil, true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "Cmd mode set",
						Command: cliffSuite.NewMessageBuilder().
							StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2", "cmd.mode.set", "fan_ctrl", "night").
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2", "evt.mode.report", "fan_ctrl", "night"),
						},
					},
					{
						Name: "Device does not exists",
						Command: cliffSuite.NewMessageBuilder().
							StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:404", "cmd.mode.set", "fan_ctrl", "night").
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectMessage("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:404", "evt.error.report", "fan_ctrl"),
						},
					},
					{
						Name: "Wrong message type",
						Command: cliffSuite.NewMessageBuilder().
							FloatMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2", "cmd.mode.set", "fan_ctrl", 1).
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2", "fan_ctrl"),
						},
					},
				},
			},
			{
				Name:     "broken set mode in controller",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(mockedfanctrl.NewController(t).
					MockSetMode("night", errors.New("broken test controller"), true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "Cmd mode set with error",
						Command: cliffSuite.NewMessageBuilder().
							StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2", "cmd.mode.set", "fan_ctrl", "night").
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2", "fan_ctrl"),
						},
					},
				},
			},
			{
				Name:     "mode outside sup_modes is still reported",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(mockedfanctrl.NewController(t).
					MockGetMode("turbo", nil, true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "Cmd mode get report",
						Command: cliffSuite.NewMessageBuilder().
							NullMessage(
								"pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2",
								"cmd.mode.get_report",
								"fan_ctrl",
							).
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2", "evt.mode.report", "fan_ctrl", "turbo"),
						},
					},
				},
			},
			{
				Name:     "broken get mode in controller",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(mockedfanctrl.NewController(t).
					MockGetMode("night", errors.New("broken test controller"), true),
				),
				Nodes: []*cliffSuite.Node{
					{
						Name: "Cmd mode get_report with error",
						Command: cliffSuite.NewMessageBuilder().
							StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2", "cmd.mode.get_report", "fan_ctrl", "night").
							Build(),
						Expectations: []*cliffSuite.Expectation{
							cliffSuite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:fan_ctrl/ad:2", "fan_ctrl"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestSpecification(t *testing.T) {
	t.Parallel()

	want := map[string]fimptype.ValueTypeT{
		fanctrl.CmdModeSet:       fimptype.VTypeString,
		fanctrl.EvtModeReport:    fimptype.VTypeString,
		fanctrl.CmdModeGetReport: fimptype.VTypeNull,
		router.EvtErrorReport:    fimptype.VTypeString,
	}

	got := make(map[string]fimptype.ValueTypeT)

	for _, intf := range fanctrl.Specification("test_adapter", "1", "2", nil, []string{"normal"}).Interfaces {
		got[intf.MsgType] = intf.ValueType
	}

	assert.Equal(t, want, got)
}

func TestSendModeReportLogsOutOfSpecModeOnlyWhenPublished(t *testing.T) { //nolint:paralleltest
	hook := test.NewGlobal()
	defer hook.Reset()

	publisher := mockedadapter.NewServicePublisher(t)
	publisher.EXPECT().PublishServiceMessage(mock.Anything, mock.Anything).Return(nil).Once()

	s := fanctrl.NewService(publisher, &fanctrl.Config{
		Specification: fanctrl.Specification("test_adapter", "1", "2", nil, []string{"normal"}),
		Controller:    mockedfanctrl.NewController(t).MockGetMode("turbo", nil, false),
	})

	sent, err := s.SendModeReport(false)
	assert.NoError(t, err)
	assert.True(t, sent)
	assert.Len(t, hook.Entries, 1)

	sent, err = s.SendModeReport(false)
	assert.NoError(t, err)
	assert.False(t, sent)
	assert.Len(t, hook.Entries, 1)
}

func routeService(controller *mockedfanctrl.Controller) cliffSuite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []cliffSuite.Mock) {
		t.Helper()

		return setupService(t, mqtt, controller)
	}
}

func setupService(t *testing.T, mqtt *fimpgo.MqttTransport, controller *mockedfanctrl.Controller) ([]*router.Routing, []*task.Task, []cliffSuite.Mock) {
	t.Helper()

	thingCfg := &adapter.ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "2",
		},
		Connector: mockedadapter.NewDefaultConnector(t),
	}

	fanCfg := &fanctrl.Config{
		Specification: fanctrl.Specification(
			"test_adapter",
			"1",
			"2",
			nil,
			[]string{"normal", "night", "away", "boost"},
		),
		Controller: controller,
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(a adapter.Adapter, p adapter.Publisher, ts adapter.ThingState) (adapter.Thing, error) {
		return adapter.NewThing(p, ts, thingCfg, fanctrl.NewService(p, fanCfg)), nil
	})
	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})
	fanctrl.RouteService(ad)

	return fanctrl.RouteService(ad), nil, nil
}
