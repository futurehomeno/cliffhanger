package adapter_test

import (
	"sync/atomic"
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

const reportingInterval = 100 * time.Millisecond

func TestTaskConnectivityReporting_StableDetails(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "task emits exactly one node report per thing when details are stable",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup: setupReportingTask(
					staticConnector(adapter.ConnectionStatusUp),
					staticConnector(adapter.ConnectionStatusUp),
				),
				Nodes: []*suite.Node{
					{
						Name:    "steady state produces exactly one report per thing across multiple ticks",
						Timeout: 500 * time.Millisecond,
						Expectations: []*suite.Expectation{
							expectNodeReport(testThingAddressB).ExactlyOnce(),
							expectNodeReport(testThingAddressC).ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestTaskConnectivityReporting_DetailsChange(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "task re-emits when connectivity details change",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup: setupReportingTask(
					togglingConnector(2),
					togglingConnector(2),
				),
				Nodes: []*suite.Node{
					{
						Name:    "changing status triggers new node reports for both things",
						Timeout: 800 * time.Millisecond,
						Expectations: []*suite.Expectation{
							expectNodeReportWithStatus(testThingAddressB, adapter.ConnectionStatusUp).AtLeastOnce(),
							expectNodeReportWithStatus(testThingAddressB, adapter.ConnectionStatusDown).AtLeastOnce(),
							expectNodeReportWithStatus(testThingAddressC, adapter.ConnectionStatusUp).AtLeastOnce(),
							expectNodeReportWithStatus(testThingAddressC, adapter.ConnectionStatusDown).AtLeastOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func expectNodeReportWithStatus(address string, status adapter.ConnectionStatus) *suite.Expectation {
	return suite.NewExpectation().
		ExpectTopic(testAdapterEvtTopic).
		ExpectType(adapter.EvtNetworkNodeReport).
		ExpectService(testAdapterName).
		Expect(router.MessageVoterFn(func(m *fimpgo.Message) bool {
			report := &adapter.ConnectivityReport{}

			if err := m.Payload.GetObjectValue(report); err != nil {
				return false
			}

			return report.Address == address && report.ConnectionStatus == status
		}))
}

type connectorBuilder func(t *testing.T) adapter.Connector

func staticConnector(status adapter.ConnectionStatus) connectorBuilder {
	return func(t *testing.T) adapter.Connector {
		t.Helper()

		c := mockedadapter.NewConnector(t)
		c.On("Connectivity").Return(&adapter.ConnectivityDetails{ConnectionStatus: status}).Maybe()

		return c
	}
}

// togglingConnector returns a connector whose Connectivity() alternates between UP and DOWN,
// flipping every `flipEvery` calls so the reporting cache sees repeated content changes.
func togglingConnector(flipEvery int32) connectorBuilder {
	return func(t *testing.T) adapter.Connector {
		t.Helper()

		var calls atomic.Int32

		c := mockedadapter.NewConnector(t)
		c.On("Connectivity").Return(func() *adapter.ConnectivityDetails {
			n := calls.Add(1)

			status := adapter.ConnectionStatusUp
			if (n-1)/flipEvery%2 == 1 {
				status = adapter.ConnectionStatusDown
			}

			return &adapter.ConnectivityDetails{ConnectionStatus: status}
		}).Maybe()

		return c
	}
}

func setupReportingTask(builderB, builderC connectorBuilder) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		connectorB := builderB(t)
		connectorC := builderC(t)

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

		return nil, adapter.TaskAdapter(ad, reportingInterval), nil
	}
}
