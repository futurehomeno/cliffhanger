package adapter_test

import (
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

const (
	testAdapterName     = fimptype.ServiceNameT("test_adapter")
	testAdapterWorkDir  = "./testdata/adapter/test_adapter"
	testAdapterCmdTopic = "pt:j1/mt:cmd/rt:ad/rn:test_adapter/ad:1"
	testAdapterEvtTopic = "pt:j1/mt:evt/rt:ad/rn:test_adapter/ad:1"
	testThingAddressB   = "2"
	testThingAddressC   = "3"
)

func TestRouteAdapter_NetworkGetNode(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "cmd.network.get_node for known address emits evt.network.node_report",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    setupAdapterWithTwoThings(),
				Nodes: []*suite.Node{
					{
						Name: "get_node for thing B",
						Command: suite.StringMessage(
							testAdapterCmdTopic,
							adapter.CmdNetworkGetNode,
							testAdapterName,
							testThingAddressB,
						),
						Expectations: []*suite.Expectation{
							expectNodeReport(testThingAddressB).ExactlyOnce(),
						},
					},
					{
						Name: "get_node for thing C",
						Command: suite.StringMessage(
							testAdapterCmdTopic,
							adapter.CmdNetworkGetNode,
							testAdapterName,
							testThingAddressC,
						),
						Expectations: []*suite.Expectation{
							expectNodeReport(testThingAddressC).ExactlyOnce(),
						},
					},
				},
			},
			{
				Name:     "cmd.network.get_node error paths",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    setupAdapterWithTwoThings(),
				Nodes: []*suite.Node{
					{
						Name: "unknown address responds with error",
						Command: suite.StringMessage(
							testAdapterCmdTopic,
							adapter.CmdNetworkGetNode,
							testAdapterName,
							"999",
						),
						Expectations: []*suite.Expectation{
							suite.ExpectError(testAdapterEvtTopic, testAdapterName).ExactlyOnce(),
							suite.NewExpectation().
								ExpectTopic(testAdapterEvtTopic).
								ExpectType(adapter.EvtNetworkNodeReport).
								ExpectService(testAdapterName).
								Never(),
						},
					},
					{
						Name: "malformed payload responds with error",
						Command: suite.IntMessage(
							testAdapterCmdTopic,
							adapter.CmdNetworkGetNode,
							testAdapterName,
							42,
						),
						Expectations: []*suite.Expectation{
							suite.ExpectError(testAdapterEvtTopic, testAdapterName).ExactlyOnce(),
							suite.NewExpectation().
								ExpectTopic(testAdapterEvtTopic).
								ExpectType(adapter.EvtNetworkNodeReport).
								ExpectService(testAdapterName).
								Never(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestRouteAdapter_NetworkGetAllNodes(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "cmd.network.get_all_nodes emits one evt.network.all_nodes_report for all things",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    setupAdapterWithTwoThings(),
				Nodes: []*suite.Node{
					{
						Name: "get_all_nodes",
						Command: suite.NullMessage(
							testAdapterCmdTopic,
							adapter.CmdNetworkGetAllNodes,
							testAdapterName,
						),
						Expectations: []*suite.Expectation{
							suite.NewExpectation().
								ExpectTopic(testAdapterEvtTopic).
								ExpectType(adapter.EvtNetworkAllNodesReport).
								ExpectService(testAdapterName).
								Expect(router.MessageVoterFn(func(m *fimpgo.Message) bool {
									var reports adapter.ConnectivityReports

									if err := m.Payload.GetObjectValue(&reports); err != nil {
										return false
									}

									return len(reports) == 2 && hasAddress(reports, testThingAddressB) && hasAddress(reports, testThingAddressC)
								})).
								ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func hasAddress(reports adapter.ConnectivityReports, addr string) bool {
	for _, r := range reports {
		if r.Address == addr {
			return true
		}
	}

	return false
}

func expectNodeReport(address string) *suite.Expectation {
	return suite.NewExpectation().
		ExpectTopic(testAdapterEvtTopic).
		ExpectType(adapter.EvtNetworkNodeReport).
		ExpectService(testAdapterName).
		Expect(router.MessageVoterFn(func(m *fimpgo.Message) bool {
			report := &adapter.ConnectivityReport{}

			if err := m.Payload.GetObjectValue(report); err != nil {
				return false
			}

			return report.Address == address
		}))
}

func setupAdapterWithTwoThings() suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		connectorB := mockedadapter.NewDefaultConnector(t)
		connectorC := mockedadapter.NewDefaultConnector(t)

		factory := adapterhelper.FactoryHelper(func(_ adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
			cfg := &adapter.ThingConfig{
				InclusionReport: &fimptype.ThingInclusionReport{Address: thingState.Address()},
			}

			switch thingState.Address() {
			case testThingAddressB:
				cfg.Connector = connectorB
			case testThingAddressC:
				cfg.Connector = connectorC
			default:
				cfg.Connector = mockedadapter.NewDefaultConnector(t)
			}

			return adapter.NewThing(publisher, thingState, cfg), nil
		})

		seeds := adapter.ThingSeeds{
			{ID: "B", CustomAddress: testThingAddressB},
			{ID: "C", CustomAddress: testThingAddressC},
		}

		ad := adapterhelper.PrepareSeededAdapter(t, testAdapterWorkDir, mqtt, factory, seeds)

		return adapter.RouteAdapter(ad), nil, nil
	}
}
