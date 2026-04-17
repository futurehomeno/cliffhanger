package diagnostic_test

import (
	"errors"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/diagnostic"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockeddiagnostic "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/diagnostic"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

var errTest = errors.New("test")

func TestRouteDiagnostic(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "successful LQI report",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockeddiagnostic.NewLQIReporter(t).
						MockLQIReport(42, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "get LQI report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "cmd.lqi.get_report", "diagnostic"),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "evt.lqi.report", "diagnostic", 42),
						},
					},
				},
			},
			{
				Name:     "successful RSSI report",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockeddiagnostic.NewRSSIReporter(t).
						MockRSSIReport(-67, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "get RSSI report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "cmd.rssi.get_report", "diagnostic"),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "evt.rssi.report", "diagnostic", -67),
						},
					},
				},
			},
			{
				Name:     "successful reboot reason report",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockeddiagnostic.NewRebootReasonReporter(t).
						MockRebootReasonReport("power_loss", nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "get reboot reason report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "cmd.reboot_reason.get_report", "diagnostic"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "evt.reboot_reason.report", "diagnostic", "power_loss"),
						},
					},
				},
			},
			{
				Name:     "successful reboots count report",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockeddiagnostic.NewRebootsCountReporter(t).
						MockRebootsCountReport(7, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "get reboots count report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "cmd.reboots_count.get_report", "diagnostic"),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "evt.reboots_count.report", "diagnostic", 7),
						},
					},
				},
			},
			{
				Name:     "successful uptime report",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockeddiagnostic.NewUptimeReporter(t).
						MockUptimeReport(3600, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "get uptime report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "cmd.uptime.get_report", "diagnostic"),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "evt.uptime.report", "diagnostic", 3600),
						},
					},
				},
			},
			{
				Name:     "successful errors report",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockeddiagnostic.NewErrorsReporter(t).
						MockErrorsReport([]string{"overcurrent", "overheat"}, nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "get errors report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "cmd.errors.get_report", "diagnostic"),
						Expectations: []*suite.Expectation{
							suite.ExpectStringArray("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "evt.errors.report", "diagnostic", []string{"overcurrent", "overheat"}),
						},
					},
				},
			},
			{
				Name:     "all reports are unsupported",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup:    routeService(mockeddiagnostic.NewController(t)),
				Nodes: []*suite.Node{
					{
						Name:    "LQI unsupported",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "cmd.lqi.get_report", "diagnostic"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "diagnostic"),
						},
					},
					{
						Name:    "RSSI unsupported",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "cmd.rssi.get_report", "diagnostic"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "diagnostic"),
						},
					},
					{
						Name:    "reboot reason unsupported",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "cmd.reboot_reason.get_report", "diagnostic"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "diagnostic"),
						},
					},
					{
						Name:    "reboots count unsupported",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "cmd.reboots_count.get_report", "diagnostic"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "diagnostic"),
						},
					},
					{
						Name:    "uptime unsupported",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "cmd.uptime.get_report", "diagnostic"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "diagnostic"),
						},
					},
					{
						Name:    "errors unsupported",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "cmd.errors.get_report", "diagnostic"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "diagnostic"),
						},
					},
				},
			},
			{
				Name:     "reporters returning errors",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					&failingReporter{},
				),
				Nodes: []*suite.Node{
					{
						Name:    "LQI report fails",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "cmd.lqi.get_report", "diagnostic"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "diagnostic"),
						},
					},
					{
						Name:    "RSSI report fails",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "cmd.rssi.get_report", "diagnostic"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "diagnostic"),
						},
					},
					{
						Name:    "reboot reason report fails",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "cmd.reboot_reason.get_report", "diagnostic"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "diagnostic"),
						},
					},
					{
						Name:    "reboots count report fails",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "cmd.reboots_count.get_report", "diagnostic"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "diagnostic"),
						},
					},
					{
						Name:    "uptime report fails",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "cmd.uptime.get_report", "diagnostic"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "diagnostic"),
						},
					},
					{
						Name:    "errors report fails",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "cmd.errors.get_report", "diagnostic"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:2", "diagnostic"),
						},
					},
					{
						Name:    "service not found under the provided address",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:3", "cmd.lqi.get_report", "diagnostic"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:diagnostic/ad:3", "diagnostic"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

// failingReporter implements every optional diagnostic reporter and returns errTest for each call.
// It lets a single setup cover error paths for all reports without juggling separate mocks.
type failingReporter struct{}

func (*failingReporter) LQIReport() (int, error)          { return 0, errTest }
func (*failingReporter) RSSIReport() (int, error)         { return 0, errTest }
func (*failingReporter) RebootReasonReport() (string, error) {
	return "", errTest
}
func (*failingReporter) RebootsCountReport() (int, error) { return 0, errTest }
func (*failingReporter) UptimeReport() (int, error)       { return 0, errTest }
func (*failingReporter) ErrorsReport() ([]string, error)  { return nil, errTest }

func routeService(controller diagnostic.Controller) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupService(t, mqtt, controller)

		return routing, nil, mocks
	}
}

func setupService(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	controller diagnostic.Controller,
) ([]*router.Routing, []*task.Task, []suite.Mock) { //nolint:unparam
	t.Helper()

	var mocks []suite.Mock
	if mockedController, ok := controller.(suite.Mock); ok {
		mocks = append(mocks, mockedController)
	}

	thingCfg := &adapter.ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "2",
		},
		Connector: mockedadapter.NewDefaultConnector(t),
	}

	diagCfg := &diagnostic.Config{
		Specification: diagnostic.Specification(
			"test_adapter",
			"1",
			"2",
			nil,
		),
		Controller: controller,
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(a adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return adapter.NewThing(publisher, thingState, thingCfg, diagnostic.NewService(publisher, diagCfg)), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return diagnostic.RouteService(ad), nil, mocks
}
