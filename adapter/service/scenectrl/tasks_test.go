package scenectrl_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter"
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

func TestTaskReportingSkipsNotReported(t *testing.T) { //nolint:paralleltest
	old := log.StandardLogger().ReplaceHooks(make(log.LevelHooks))
	defer log.StandardLogger().ReplaceHooks(old)

	hook := logtest.NewGlobal()

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Scene not reported yet",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskSceneCtrl(
					mockedscenectrl.NewController(t).
						MockSceneCtrlSceneReport(scenectrl.SceneReport{}, errTest, true).
						MockSceneCtrlSceneReport(scenectrl.SceneReport{}, fmt.Errorf("no button event: %w", adapter.ErrNotReported), false),
					[]string{"scene1", "scene2"},
					50*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name:    "No report is sent",
						Timeout: 300 * time.Millisecond,
						Expectations: []*suite.Expectation{
							suite.ExpectMessage("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "evt.scene.report", "scene_ctrl").Never(),
						},
					},
				},
			},
		},
	}

	s.Run(t)

	logged := 0

	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, "[scenectrl]") {
			logged++
		}
	}

	assert.Equal(t, 1, logged, "only the other error is logged, never the state not reported yet")
}
