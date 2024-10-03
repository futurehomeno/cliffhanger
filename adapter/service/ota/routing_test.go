package ota_test

import (
	"errors"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/ota"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedota "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/ota"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

const (
	testFirmwarePath = "test/firmware/path"
)

func TestRouteService(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "OTA update start - no controller errors",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedota.NewController(t).MockStartOTAUpdate(testFirmwarePath, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Commands: []*fimpgo.Message{suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:ota/ad:1", "cmd.ota_update.start", "ota", testFirmwarePath)},
						Expectations: []*suite.Expectation{
							suite.ExpectNull("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:ota/ad:1", "evt.ota_start.report", "ota").Never(),
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:ota/ad:1", "ota").Never(),
						},
					},
				},
			},
			{
				Name:     "OTA update start - controller error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedota.NewController(t).MockStartOTAUpdate(testFirmwarePath, errors.New("oops"), true),
				),
				Nodes: []*suite.Node{
					{
						Commands: []*fimpgo.Message{suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:ota/ad:1", "cmd.ota_update.start", "ota", testFirmwarePath)},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:ota/ad:1", "ota"),
						},
					},
				},
			},
			{
				Name:     "OTA update start - error - invalid message type",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedota.NewController(t),
				),
				Nodes: []*suite.Node{
					{
						Commands: []*fimpgo.Message{suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:ota/ad:1", "cmd.ota_update.start", "ota", 1)},
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:ota/ad:1", "ota"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestOTAReporting(t *testing.T) { //nolint:paralleltest
	var otaService ota.Service

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "report start",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: setupOTAService(&otaService, mockedota.NewController(t).
					MockOTAStatusReport(
						ota.StatusReport{
							Status: ota.StatusStarted,
						},
						nil,
						true,
					),
				),
				Nodes: []*suite.Node{
					{
						Callbacks: []suite.Callback{sendStatusReportCallback(&otaService, false)},
						Expectations: []*suite.Expectation{
							suite.ExpectNull("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:ota/ad:1", "evt.ota_start.report", "ota"),
						},
					},
				},
			},
			{
				Name:     "report progress only",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: setupOTAService(&otaService, mockedota.NewController(t).
					MockOTAStatusReport(
						ota.StatusReport{
							Status: ota.StatusInProgress,
							Progress: ota.ProgressData{
								Progress: 20,
							},
						},
						nil,
						true,
					),
				),
				Nodes: []*suite.Node{
					{
						Callbacks: []suite.Callback{sendStatusReportCallback(&otaService, false)},
						Expectations: []*suite.Expectation{
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:ota/ad:1", "evt.ota_progress.report", "ota", map[string]int64{
								"progress": 20,
							}),
						},
					},
				},
			},
			{
				Name:     "report progress and remaining minutes",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: setupOTAService(&otaService, mockedota.NewController(t).
					MockOTAStatusReport(
						ota.StatusReport{
							Status: ota.StatusInProgress,
							Progress: ota.ProgressData{
								Progress:         20,
								RemainingMinutes: 2,
							},
						},
						nil,
						true,
					),
				),
				Nodes: []*suite.Node{
					{
						Callbacks: []suite.Callback{sendStatusReportCallback(&otaService, false)},
						Expectations: []*suite.Expectation{
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:ota/ad:1", "evt.ota_progress.report", "ota", map[string]int64{
								"progress":      20,
								"remaining_min": 2,
							}),
						},
					},
				},
			},
			{
				Name:     "full progress report",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: setupOTAService(&otaService, mockedota.NewController(t).
					MockOTAStatusReport(
						ota.StatusReport{
							Status: ota.StatusInProgress,
							Progress: ota.ProgressData{
								Progress:         20,
								RemainingMinutes: 2,
								RemainingSeconds: 32,
							},
						},
						nil,
						true,
					),
				),
				Nodes: []*suite.Node{
					{
						Callbacks: []suite.Callback{sendStatusReportCallback(&otaService, false)},
						Expectations: []*suite.Expectation{
							suite.ExpectIntMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:ota/ad:1", "evt.ota_progress.report", "ota", map[string]int64{
								"progress":      20,
								"remaining_min": 2,
								"remaining_sec": 32,
							}),
						},
					},
				},
			},
			{
				Name:     "idle status - do nothing",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: setupOTAService(&otaService, mockedota.NewController(t).
					MockOTAStatusReport(
						ota.StatusReport{
							Status: ota.StatusIdle,
						},
						nil,
						true,
					),
				),
				Nodes: []*suite.Node{
					{
						Callbacks: []suite.Callback{sendStatusReportCallback(&otaService, false)},
						Expectations: []*suite.Expectation{
							suite.ExpectMessage("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:ota/ad:1", "evt.ota_progress.report", "ota").Never(),
						},
					},
				},
			},
			{
				Name:     "end report with no errors",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: setupOTAService(&otaService, mockedota.NewController(t).
					MockOTAStatusReport(
						ota.StatusReport{
							Status: ota.StatusDone,
							Result: ota.ResultData{
								Error: "",
							},
						},
						nil,
						true,
					),
				),
				Nodes: []*suite.Node{
					{
						Callbacks: []suite.Callback{sendStatusReportCallback(&otaService, false)},
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:ota/ad:1", "evt.ota_end.report", "ota", ota.EndReport{
								Success: true,
								Error:   "",
							}),
						},
					},
				},
			},
			{
				Name:     "end report with errors",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: setupOTAService(&otaService, mockedota.NewController(t).
					MockOTAStatusReport(
						ota.StatusReport{
							Status: ota.StatusDone,
							Result: ota.ResultData{
								Error: ota.ErrInvalidImage,
							},
						},
						nil,
						true,
					),
				),
				Nodes: []*suite.Node{
					{
						Callbacks: []suite.Callback{sendStatusReportCallback(&otaService, false)},
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:ota/ad:1", "evt.ota_end.report", "ota", ota.EndReport{
								Success: false,
								Error:   "invalid_image",
							}),
						},
					},
				},
			},
			{
				Name:     "controller error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: setupOTAService(&otaService, mockedota.NewController(t).
					MockOTAStatusReport(ota.StatusReport{}, errors.New("oops"), true),
				),
				Nodes: []*suite.Node{
					{
						Callbacks: []suite.Callback{sendStatusReportCallback(&otaService, true)},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeService(controller ota.Controller) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		mockedController, ok := controller.(suite.Mock)
		if !ok {
			t.Fatal("controller is not a mock")
		}

		mocks := []suite.Mock{mockedController}
		ad := setupAdapter(t, mqtt, controller)

		return ota.RouteService(ad), nil, mocks
	}
}

func setupOTAService(service *ota.Service, controller ota.Controller) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) (_ []*router.Routing, _ []*task.Task, _ []suite.Mock) {
		t.Helper()

		ad := setupAdapter(t, mqtt, controller)

		ss := ad.Services(ota.OTA)
		if len(ss) != 1 {
			t.Fatalf("expected 1 service, got %d", len(ss))
		}

		s, ok := ss[0].(ota.Service)
		if !ok {
			t.Fatal("service is not an OTA service")
		}

		*service = s

		return nil, nil, nil
	}
}

func setupAdapter(t *testing.T, mqtt *fimpgo.MqttTransport, controller ota.Controller) adapter.Adapter {
	t.Helper()

	thingCfg := &adapter.ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "1",
		},
		Connector: mockedadapter.NewDefaultConnector(t),
	}

	serviceCfg := &ota.Config{
		Specification: ota.Specification(
			"test_adapter",
			"1",
			"1",
			nil,
		),
		Controller: controller,
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "1"}

	factory := adapterhelper.FactoryHelper(func(_ adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return adapter.NewThing(publisher, thingState, thingCfg, ota.NewService(publisher, serviceCfg)), nil
	})

	return adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})
}

func sendStatusReportCallback(s *ota.Service, wantErr bool) suite.Callback {
	return func(t *testing.T) {
		t.Helper()

		service := *s

		err := service.SendStatusReport()
		if wantErr {
			require.Error(t, err)

			return
		}

		require.NoError(t, err)
	}
}
