package scenectrl_test

//import (
//	"errors"
//	"testing"
//	"time"
//
//	"github.com/futurehomeno/fimpgo"
//	"github.com/futurehomeno/fimpgo/fimptype"
//
//	"github.com/futurehomeno/cliffhanger/adapter"
//	"github.com/futurehomeno/cliffhanger/adapter/service/scenectrl"
//	"github.com/futurehomeno/cliffhanger/event"
//	"github.com/futurehomeno/cliffhanger/router"
//	"github.com/futurehomeno/cliffhanger/task"
//	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
//	mockedscenectrl "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/scenectrl"
//	"github.com/futurehomeno/cliffhanger/test/suite"
//)
//
//const (
//	sceneNone        = "none"
//	sceneColorloop   = "colorloop"
//	sceneUnsupported = "movietime"
//)
//
//func TestRouteSceneCtrl(t *testing.T) { //nolint:paralleltest
//	sceneReportColorloop := scenectrl.SceneReport{
//		Scene:     sceneColorloop,
//		Timestamp: time.Now(),
//	}
//
//	s := &suite.Suite{
//		Cases: []*suite.Case{
//			{
//				Name: "successful set scene",
//				Setup: routeSceneCtrl(
//					mockedscenectrl.NewController(t).
//						MockSetSceneCtrlScene(sceneColorloop, nil, true).
//						MockSceneCtrlSceneReport(sceneReportColorloop, nil, true),
//				),
//				Nodes: []*suite.Node{
//					{
//						Name:    "set scene",
//						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "cmd.scene.set", "scene_ctrl", sceneColorloop),
//						Expectations: []*suite.Expectation{
//							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "evt.scene.report", "scene_ctrl", sceneColorloop),
//						},
//					},
//				},
//			},
//			{
//				Name: "successful get scene",
//				Setup: routeSceneCtrl(
//					mockedscenectrl.NewController(t).
//						MockSceneCtrlSceneReport(sceneReportColorloop, nil, true),
//				),
//				Nodes: []*suite.Node{
//					{
//						Name:    "get scene",
//						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "cmd.scene.get_report", "scene_ctrl"),
//						Expectations: []*suite.Expectation{
//							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "evt.scene.report", "scene_ctrl", sceneColorloop),
//						},
//					},
//				},
//			},
//			{
//				Name: "failed set scene - setting error",
//				Setup: routeSceneCtrl(
//					mockedscenectrl.NewController(t).
//						MockSetSceneCtrlScene(sceneColorloop, errors.New("error"), true),
//				),
//				Nodes: []*suite.Node{
//					{
//						Name:    "controller error",
//						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "cmd.scene.set", "scene_ctrl", sceneColorloop),
//						Expectations: []*suite.Expectation{
//							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "scene_ctrl"),
//						},
//					},
//					{
//						Name:    "invalid value type",
//						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "cmd.scene.set", "scene_ctrl", 1),
//						Expectations: []*suite.Expectation{
//							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "scene_ctrl"),
//						},
//					},
//					{
//						Name:    "wrong address",
//						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:3", "cmd.scene.set", "scene_ctrl", sceneColorloop),
//						Expectations: []*suite.Expectation{
//							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:3", "scene_ctrl"),
//						},
//					},
//					{
//						Name:    "unsupported scene",
//						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "cmd.scene.set", "scene_ctrl", sceneUnsupported),
//						Expectations: []*suite.Expectation{
//							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "scene_ctrl"),
//						},
//					},
//				},
//			},
//			{
//				Name: "failed set scene - report error",
//				Setup: routeSceneCtrl(
//					mockedscenectrl.NewController(t).
//						MockSetSceneCtrlScene(sceneColorloop, nil, true).
//						MockSceneCtrlSceneReport(sceneReportColorloop, errors.New("error"), true),
//				),
//				Nodes: []*suite.Node{
//					{
//						Name:    "report error",
//						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "cmd.scene.set", "scene_ctrl", sceneColorloop),
//						Expectations: []*suite.Expectation{
//							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "scene_ctrl"),
//						},
//					},
//				},
//			},
//		},
//	}
//
//	s.Run(t)
//}
//
//func TestTaskSceneCtrl(t *testing.T) { //nolint:paralleltest
//	sceneReport1 := scenectrl.SceneReport{
//		Scene:     sceneColorloop,
//		Timestamp: time.Date(2022, time.January, 1, 1, 1, 1, 1, time.UTC),
//	}
//
//	sceneReport2 := scenectrl.SceneReport{
//		Scene:     sceneColorloop,
//		Timestamp: time.Date(2022, time.January, 2, 2, 2, 2, 2, time.UTC),
//	}
//
//	sceneReport3 := scenectrl.SceneReport{
//		Scene:     sceneNone,
//		Timestamp: time.Date(2022, time.January, 2, 2, 2, 2, 2, time.UTC),
//	}
//
//	s := &suite.Suite{
//		Cases: []*suite.Case{
//			{
//				Name: "SceneCtrl Tasks",
//				Setup: taskSceneCtrl(
//					mockedscenectrl.NewController(t).
//						MockSceneCtrlSceneReport(sceneReport1, nil, true).
//						MockSceneCtrlSceneReport(sceneReport2, nil, true).
//						MockSceneCtrlSceneReport(sceneReport3, nil, true).
//						MockSceneCtrlSceneReport(scenectrl.SceneReport{}, errors.New("task error"), false),
//					100*time.Millisecond,
//				),
//				Nodes: []*suite.Node{
//					{
//						Name: "Tasks",
//						Expectations: []*suite.Expectation{
//							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "evt.scene.report", "scene_ctrl", sceneColorloop),
//							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:scene_ctrl/ad:2", "evt.scene.report", "scene_ctrl", sceneNone),
//						},
//					},
//				},
//			},
//		},
//	}
//
//	s.Run(t)
//}
//
//func routeSceneCtrl(
//	sceneControlController *mockedscenectrl.Controller,
//) suite.BaseSetup {
//	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
//		t.Helper()
//
//		routing, _, mocks := setupSceneCtrl(t, mqtt, sceneControlController, 0)
//
//		return routing, nil, mocks
//	}
//}
//
//func taskSceneCtrl(
//	sceneCtrlController *mockedscenectrl.Controller,
//	interval time.Duration,
//) suite.BaseSetup {
//	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
//		t.Helper()
//
//		routing, tasks, mocks := setupSceneCtrl(t, mqtt, sceneCtrlController, interval)
//
//		return routing, tasks, mocks
//	}
//}
//
//func setupSceneCtrl(
//	t *testing.T,
//	mqtt *fimpgo.MqttTransport,
//	sceneCtrlController *mockedscenectrl.Controller,
//	interval time.Duration,
//) ([]*router.Routing, []*task.Task, []suite.Mock) {
//	t.Helper()
//
//	mocks := []suite.Mock{sceneCtrlController}
//
//	cfg := &sceneCtrlThingConfig{
//		ThingConfig: &adapter.ThingConfig{
//			InclusionReport: &fimptype.ThingInclusionReport{
//				Address: "2",
//			},
//			Connector: mockedadapter.NewConnector(t),
//		},
//		SceneCtrlConfig: &scenectrl.Config{
//			Specification: scenectrl.Specification(
//				"test_adapter",
//				"1",
//				"2",
//				[]string{sceneNone, sceneColorloop},
//				nil,
//			),
//			Controller: sceneCtrlController,
//		},
//	}
//
//	ad := adapter.NewAdapter(mqtt, event.NewManager(), nil, nil, "test_adapter", "1")
//
//	sceneCtrlThing := newSceneCtrlThing(ad, cfg)
//
//	ad.RegisterThing(sceneCtrlThing)
//
//	return routeSceneCtrlThing(ad), taskSceneCtrlThing(ad, interval), mocks
//}
//
//// sceneCtrlThingConfig represents a config for testing scene control service.
//type sceneCtrlThingConfig struct {
//	ThingConfig     *adapter.ThingConfig
//	SceneCtrlConfig *scenectrl.Config
//}
//
//// newSceneCtrlThing creates a thing that can be used for testing scene control service.
//func newSceneCtrlThing(
//	publisher adapter.Publisher,
//	cfg *sceneCtrlThingConfig,
//) adapter.Thing {
//	services := []adapter.Service{
//		scenectrl.NewService(a, cfg.SceneCtrlConfig),
//	}
//
//	return adapter.NewThing(a, nil, cfg.ThingConfig, services...)
//}
//
//// routeSceneCtrlThing creates a thing that can be used for testing scene control service.
//func routeSceneCtrlThing(ad adapter.Adapter) []*router.Routing {
//	return router.Combine(
//		scenectrl.RouteService(ad),
//	)
//}
//
//// taskSceneCtrlThing creates background tasks specific for a scene control service.
//func taskSceneCtrlThing(
//	ad adapter.Adapter,
//	interval time.Duration,
//	voter ...task.Voter,
//) []*task.Task {
//	return []*task.Task{
//		scenectrl.TaskReporting(ad, interval, voter...),
//	}
//}
