package scenectrl_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter/service/scenectrl"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedscenectrl "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/scenectrl"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestTaskReporting(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Scene controller tasks",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskSceneCtrl(
					mockedscenectrl.NewController(t).
						MockSceneCtrlSceneReport(scenectrl.SceneReport{Scene: "scene1"}, nil, true).
						MockSceneCtrlSceneReport(scenectrl.SceneReport{}, errTest, true).
						MockSceneCtrlSceneReport(scenectrl.SceneReport{Scene: "scene1"}, nil, true).
						MockSceneCtrlSceneReport(scenectrl.SceneReport{Scene: "scene2"}, nil, true), // should be sent twice
					[]string{"scene1", "scene2"},
					100*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "One change and one error during four report cycles",
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "evt.scene.report", "scene_ctrl", "scene1").ExactlyOnce(),
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "evt.scene.report", "scene_ctrl", "scene2").ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func taskSceneCtrl(
	controller scenectrl.Controller,
	supportedScenes []string,
	interval time.Duration,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		_, tasks, mocks := setupService(t, mqtt, controller, supportedScenes, interval)

		return nil, tasks, mocks
	}
}
