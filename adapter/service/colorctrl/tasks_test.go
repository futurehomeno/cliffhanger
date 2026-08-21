package colorctrl_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter/service/colorctrl"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedcolorctrl "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/colorctrl"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestTaskReporting(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Color ctrl tasks",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskColorCtrl(
					mockedcolorctrl.NewController(t).
						MockColorCtrlColorReport(map[string]int{"red": 255, "green": 0, "blue": 0}, nil, true).
						MockColorCtrlColorReport(nil, errTest, true).
						MockColorCtrlColorReport(map[string]int{"red": 255, "green": 0, "blue": 0}, nil, true).
						MockColorCtrlColorReport(map[string]int{"red": 0, "green": 255, "blue": 0}, nil, true), // should be sent twice
					100*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "One change and one error during four report cycles",
						Expectations: []*suite.Expectation{
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "evt.color.report", "color_ctrl", map[string]int{"red": 255, "green": 0, "blue": 0}).ExactlyOnce(),
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:color_ctrl/ad:2", "evt.color.report", "color_ctrl", map[string]int{"red": 0, "green": 255, "blue": 0}).ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func taskColorCtrl(
	controller colorctrl.Controller,
	interval time.Duration,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		_, tasks, mocks := setupService(t, mqtt, controller, interval)

		return nil, tasks, mocks
	}
}
