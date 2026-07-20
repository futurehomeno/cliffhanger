package adapter_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

// seedByID maps a device to a seed whose address equals its ID, so sync-created things land
// at predictable addresses the report matchers can assert on.
func seedByID(d device) *adapter.ThingSeed {
	return &adapter.ThingSeed{ID: d.ID, CustomAddress: d.ID, Info: d.Name}
}

func syncReportSetup(ad *adapter.Adapter, seeds adapter.ThingSeeds) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		factory := adapterhelper.FactoryHelper(func(_ adapter.Adapter, publisher adapter.Publisher, ts adapter.ThingState) (adapter.Thing, error) {
			cfg := &adapter.ThingConfig{
				InclusionReport: &fimptype.ThingInclusionReport{Address: ts.Address()},
				Connector:       mockedadapter.NewDefaultConnector(t),
			}

			return adapter.NewThing(publisher, ts, cfg), nil
		})

		*ad = adapterhelper.PrepareSeededAdapter(t, testAdapterWorkDir, mqtt, factory, seeds)

		return adapter.RouteAdapter(*ad), nil, nil
	}
}

func TestSyncThings_AddsDeviceAndSendsInclusionReport(t *testing.T) { //nolint:paralleltest
	var ad adapter.Adapter

	// Three devices are already registered; a fourth appears in the fetched list.
	seeds := adapter.ThingSeeds{seedByID(device{ID: "1"}), seedByID(device{ID: "2"}), seedByID(device{ID: "3"})}
	available := []device{{"1", "a"}, {"2", "b"}, {"3", "c"}, {"4", "d"}}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "a new fetched device is added and announced",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    syncReportSetup(&ad, seeds),
				Nodes: []*suite.Node{
					{
						Name:    "syncing four devices over three creates the fourth and sends its inclusion report",
						Timeout: 500 * time.Millisecond,
						InitCallbacks: []suite.Callback{func(t *testing.T) {
							t.Helper()

							assert.Nil(t, ad.ThingByID("4"), "device 4 must not exist before the sync")
							assert.NoError(t, adapter.SyncThings(ad, available, []string{"1", "2", "3", "4"}, seedByID))
							assert.NotNil(t, ad.ThingByID("4"), "device 4 must be registered after the sync")
						}},
						Expectations: []*suite.Expectation{
							expectInclusion("4").AtLeastOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestSyncThings_RemovesDeviceAndSendsExclusionReport(t *testing.T) { //nolint:paralleltest
	var ad adapter.Adapter

	// Four devices are registered; the fetched list drops the fourth (genuinely deselected).
	seeds := adapter.ThingSeeds{
		seedByID(device{ID: "1"}), seedByID(device{ID: "2"}), seedByID(device{ID: "3"}), seedByID(device{ID: "4"}),
	}
	available := []device{{"1", "a"}, {"2", "b"}, {"3", "c"}}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "a dropped device is destroyed and announced",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    syncReportSetup(&ad, seeds),
				Nodes: []*suite.Node{
					{
						Name:    "syncing three devices over four destroys the fourth and sends its exclusion report",
						Timeout: 500 * time.Millisecond,
						InitCallbacks: []suite.Callback{func(t *testing.T) {
							t.Helper()

							assert.NotNil(t, ad.ThingByID("4"), "device 4 must exist before the sync")
							assert.NoError(t, adapter.SyncThings(ad, available, []string{"1", "2", "3"}, seedByID))
							assert.Nil(t, ad.ThingByID("4"), "device 4 must be removed after the sync")
						}},
						Expectations: []*suite.Expectation{
							expectExclusion("4").ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestEnsureThings_AddsDeviceAndSendsInclusionReport(t *testing.T) { //nolint:paralleltest
	var ad adapter.Adapter

	// Three devices registered; EnsureThings is called directly with a fourth seed added.
	seeds := adapter.ThingSeeds{seedByID(device{ID: "1"}), seedByID(device{ID: "2"}), seedByID(device{ID: "3"})}
	withFourth := adapter.ThingSeeds{
		seedByID(device{ID: "1"}), seedByID(device{ID: "2"}), seedByID(device{ID: "3"}), seedByID(device{ID: "4"}),
	}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "EnsureThings creates a missing thing and announces it",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    syncReportSetup(&ad, seeds),
				Nodes: []*suite.Node{
					{
						Name:    "an extra seed creates the fourth thing and sends its inclusion report",
						Timeout: 500 * time.Millisecond,
						InitCallbacks: []suite.Callback{func(t *testing.T) {
							t.Helper()

							assert.Nil(t, ad.ThingByID("4"), "device 4 must not exist before EnsureThings")
							assert.NoError(t, ad.EnsureThings(withFourth))
							assert.NotNil(t, ad.ThingByID("4"), "device 4 must be registered after EnsureThings")
						}},
						Expectations: []*suite.Expectation{
							expectInclusion("4").AtLeastOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestEnsureThings_RemovesDeviceAndSendsExclusionReport(t *testing.T) { //nolint:paralleltest
	var ad adapter.Adapter

	// Four devices registered; EnsureThings is called directly with the fourth seed dropped.
	seeds := adapter.ThingSeeds{
		seedByID(device{ID: "1"}), seedByID(device{ID: "2"}), seedByID(device{ID: "3"}), seedByID(device{ID: "4"}),
	}
	withoutFourth := adapter.ThingSeeds{seedByID(device{ID: "1"}), seedByID(device{ID: "2"}), seedByID(device{ID: "3"})}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "EnsureThings destroys a stale thing and announces it",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    syncReportSetup(&ad, seeds),
				Nodes: []*suite.Node{
					{
						Name:    "a dropped seed destroys the fourth thing and sends its exclusion report",
						Timeout: 500 * time.Millisecond,
						InitCallbacks: []suite.Callback{func(t *testing.T) {
							t.Helper()

							assert.NotNil(t, ad.ThingByID("4"), "device 4 must exist before EnsureThings")
							assert.NoError(t, ad.EnsureThings(withoutFourth))
							assert.Nil(t, ad.ThingByID("4"), "device 4 must be removed after EnsureThings")
						}},
						Expectations: []*suite.Expectation{
							expectExclusion("4").ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func expectInclusion(address string) *suite.Expectation {
	return suite.NewExpectation().
		ExpectTopic(testAdapterEvtTopic).
		ExpectType(adapter.EvtThingInclusionReport).
		ExpectService(testAdapterName).
		Expect(router.MessageVoterFn(func(m *fimpgo.Message) bool {
			var report fimptype.ThingInclusionReport

			if err := m.Payload.GetObjectValue(&report); err != nil {
				return false
			}

			return report.Address == address
		}))
}
