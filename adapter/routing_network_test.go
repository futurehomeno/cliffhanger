package adapter_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func expectAdapterEvent(msgType string, match func(m *fimpgo.Message) bool) *suite.Expectation {
	return suite.NewExpectation().
		ExpectTopic(testAdapterEvtTopic).
		ExpectType(msgType).
		ExpectService(testAdapterName).
		Expect(router.MessageVoterFn(match))
}

func TestRouteAdapter_NetworkReset(t *testing.T) { //nolint:paralleltest
	reset := suite.NullMessage(testAdapterCmdTopic, adapter.CmdNetworkReset, testAdapterName)

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "cmd.network.reset excludes every thing and confirms with evt.network.reset_done",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    setupAdapterWithTwoThings(),
				Nodes: []*suite.Node{
					{
						Name:    "reset",
						Command: reset,
						Expectations: []*suite.Expectation{
							expectExclusion(testThingAddressB).ExactlyOnce(),
							expectExclusion(testThingAddressC).ExactlyOnce(),
							expectAdapterEvent(adapter.EvtNetworkResetDone, func(m *fimpgo.Message) bool {
								return m.Payload.CorrelationID == reset.Payload.UID
							}).ExactlyOnce(),
							suite.ExpectError(testAdapterEvtTopic, testAdapterName).Never(),
						},
					},
					{
						Name:    "the reset thing is gone",
						Command: suite.StringMessage(testAdapterCmdTopic, adapter.CmdNetworkGetNode, testAdapterName, testThingAddressB),
						Expectations: []*suite.Expectation{
							suite.ExpectError(testAdapterEvtTopic, testAdapterName).ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestRouteAdapter_ThingGetInclusionReport(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "cmd.thing.get_inclusion_report forces the report",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    setupAdapterWithTwoThings(),
				Nodes: []*suite.Node{
					{
						Name:    "an unchanged report is still sent",
						Command: suite.StringMessage(testAdapterCmdTopic, adapter.CmdThingGetInclusionReport, testAdapterName, testThingAddressB),
						Expectations: []*suite.Expectation{
							expectInclusion(testThingAddressB).ExactlyOnce(),
						},
					},
					{
						Name:    "unknown address responds with error",
						Command: suite.StringMessage(testAdapterCmdTopic, adapter.CmdThingGetInclusionReport, testAdapterName, "999"),
						Expectations: []*suite.Expectation{
							suite.ExpectError(testAdapterEvtTopic, testAdapterName).ExactlyOnce(),
						},
					},
					{
						Name: "a command addressed to another adapter is ignored",
						Command: suite.StringMessage(
							"pt:j1/mt:cmd/rt:ad/rn:other_adapter/ad:1",
							adapter.CmdThingGetInclusionReport,
							"other_adapter",
							testThingAddressB,
						),
						Timeout: 500 * time.Millisecond,
						Expectations: []*suite.Expectation{
							expectInclusion(testThingAddressB).Never(),
							suite.ExpectError(testAdapterEvtTopic, testAdapterName).Never(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestRouteAdapter_PingSend(t *testing.T) { //nolint:paralleltest
	setup := suite.BaseSetup(func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		factory := adapterhelper.FactoryHelper(func(_ adapter.Adapter, publisher adapter.Publisher, ts adapter.ThingState) (adapter.Thing, error) {
			connector := mockedadapter.NewDefaultConnector(t)
			connector.On("Ping").Return(&adapter.PingDetails{Status: adapter.PingResultSuccess}).Maybe()

			return adapter.NewThing(publisher, ts, &adapter.ThingConfig{
				InclusionReport: &fimptype.ThingInclusionReport{Address: ts.Address()},
				Connector:       connector,
			}), nil
		})

		ad := adapterhelper.PrepareSeededAdapter(t, testAdapterWorkDir, mqtt, factory, adapter.ThingSeeds{{ID: "B", CustomAddress: testThingAddressB}})

		return adapter.RouteAdapter(ad), nil, nil
	})

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "cmd.ping.send pings the thing and reports the result",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    setup,
				Nodes: []*suite.Node{
					{
						Name:    "ping a known thing",
						Command: suite.StringMessage(testAdapterCmdTopic, adapter.CmdPingSend, testAdapterName, testThingAddressB),
						Expectations: []*suite.Expectation{
							expectAdapterEvent(adapter.EvtPingReport, func(m *fimpgo.Message) bool {
								report := &adapter.PingReport{}

								return m.Payload.GetObjectValue(report) == nil &&
									report.Address == testThingAddressB &&
									report.PingDetails != nil && report.Status == adapter.PingResultSuccess
							}).ExactlyOnce(),
						},
					},
					{
						Name:    "ping an unknown thing",
						Command: suite.StringMessage(testAdapterCmdTopic, adapter.CmdPingSend, testAdapterName, "999"),
						Expectations: []*suite.Expectation{
							suite.ExpectError(testAdapterEvtTopic, testAdapterName).ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}
