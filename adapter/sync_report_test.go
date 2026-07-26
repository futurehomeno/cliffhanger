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

							_, err := adapter.SyncThings(ad, fetchOK(available), []string{"1", "2", "3", "4"}, seedByID)

							assert.NoError(t, err)
							assert.NotNil(t, ad.ThingByID("4"), "device 4 must be registered after the sync")
						}},
						Expectations: []*suite.Expectation{
							expectInclusion("4").ExactlyOnce(),
							expectAnyExclusion().Never(),
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

							_, err := adapter.SyncThings(ad, fetchOK(available), []string{"1", "2", "3"}, seedByID)

							assert.NoError(t, err)
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

func TestSyncThings_FetchFailureKeepsThings(t *testing.T) { //nolint:paralleltest
	var ad adapter.Adapter

	// The device list cannot be fetched, e.g. the third party is down. Nothing may be touched:
	// this is the guarantee that replaced the old refuse-to-reconcile guard.
	seeds := adapter.ThingSeeds{seedByID(device{ID: "1"}), seedByID(device{ID: "2"}), seedByID(device{ID: "3"})}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "a failed fetch destroys nothing",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    syncReportSetup(&ad, seeds),
				Nodes: []*suite.Node{
					{
						Name:    "syncing with a failing fetch leaves every thing registered and silent",
						Timeout: 500 * time.Millisecond,
						InitCallbacks: []suite.Callback{func(t *testing.T) {
							t.Helper()

							_, err := adapter.SyncThings(ad, fetchErr[device](errFetch), []string{"1", "2", "3"}, seedByID)

							assert.ErrorIs(t, err, errFetch)
							assert.NotNil(t, ad.ThingByID("1"), "device 1 must survive a failed fetch")
							assert.NotNil(t, ad.ThingByID("2"), "device 2 must survive a failed fetch")
							assert.NotNil(t, ad.ThingByID("3"), "device 3 must survive a failed fetch")
						}},
						Expectations: []*suite.Expectation{
							expectAnyExclusion().Never(),
							expectAnyInclusion().Never(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestSyncThings_EmptyFetchDestroysSelectedThings(t *testing.T) { //nolint:paralleltest
	var ad adapter.Adapter

	// The fetch succeeded and reports no devices, so the account genuinely has none left. The
	// inverse of the test above: a successful response is authoritative even when it is empty.
	seeds := adapter.ThingSeeds{seedByID(device{ID: "1"}), seedByID(device{ID: "2"}), seedByID(device{ID: "3"})}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "an empty successful fetch destroys every selected thing",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    syncReportSetup(&ad, seeds),
				Nodes: []*suite.Node{
					{
						Name:    "syncing an empty device list excludes all three things",
						Timeout: 500 * time.Millisecond,
						InitCallbacks: []suite.Callback{func(t *testing.T) {
							t.Helper()

							_, err := adapter.SyncThings(ad, fetchOK([]device{}), []string{"1", "2", "3"}, seedByID)

							assert.NoError(t, err)
							assert.Nil(t, ad.ThingByID("1"), "device 1 must be removed")
							assert.Nil(t, ad.ThingByID("2"), "device 2 must be removed")
							assert.Nil(t, ad.ThingByID("3"), "device 3 must be removed")
						}},
						Expectations: []*suite.Expectation{
							expectExclusion("1").ExactlyOnce(),
							expectExclusion("2").ExactlyOnce(),
							expectExclusion("3").ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestSyncThings_ExcludesVanishedUnownedDevice(t *testing.T) { //nolint:paralleltest
	var ad adapter.Adapter

	// Device "legacy" is selected but the adapter owns no thing for it - the state a hub is in
	// after migrating from an adapter that predates the thing store. Its stale node must still
	// be cleared, while the owned device is left untouched.
	seeds := adapter.ThingSeeds{seedByID(device{ID: "1"})}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "a vanished device the adapter does not own is still excluded",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    syncReportSetup(&ad, seeds),
				Nodes: []*suite.Node{
					{
						Name:    "syncing without the legacy device excludes it at its own ID",
						Timeout: 500 * time.Millisecond,
						InitCallbacks: []suite.Callback{func(t *testing.T) {
							t.Helper()

							_, err := adapter.SyncThings(ad, fetchOK([]device{{"1", "a"}}), []string{"1", "legacy"}, seedByID)

							assert.NoError(t, err)
							assert.NotNil(t, ad.ThingByID("1"), "the owned device must survive")
						}},
						Expectations: []*suite.Expectation{
							expectExclusion("legacy").ExactlyOnce(),
							expectExclusion("1").Never(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestEnsureThings_ContinuesAfterFactoryError(t *testing.T) { //nolint:paralleltest
	var ad adapter.Adapter

	// One device cannot be built. Best effort means the other two are still created, and the
	// broken one leaves no state record behind, so a later attempt retries it rather than
	// treating it as present forever.
	setup := suite.BaseSetup(func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		factory := adapterhelper.FactoryHelper(func(_ adapter.Adapter, publisher adapter.Publisher, ts adapter.ThingState) (adapter.Thing, error) {
			if ts.ID() == "bad" {
				return nil, errFactory
			}

			cfg := &adapter.ThingConfig{
				InclusionReport: &fimptype.ThingInclusionReport{Address: ts.Address()},
				Connector:       mockedadapter.NewDefaultConnector(t),
			}

			return adapter.NewThing(publisher, ts, cfg), nil
		})

		ad = adapterhelper.SeedAdapter(t, adapterhelper.PrepareAdapter(t, testAdapterWorkDir, mqtt, factory), nil)

		return adapter.RouteAdapter(ad), nil, nil
	})

	all := adapter.ThingSeeds{seedByID(device{ID: "good1"}), seedByID(device{ID: "bad"}), seedByID(device{ID: "good2"})}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "a device that cannot be built does not block the others",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    setup,
				Nodes: []*suite.Node{
					{
						Name:    "the two healthy devices are created and announced",
						Timeout: 500 * time.Millisecond,
						InitCallbacks: []suite.Callback{func(t *testing.T) {
							t.Helper()

							err := ad.EnsureThings(all)

							assert.ErrorIs(t, err, errFactory)
							assert.NotNil(t, ad.ThingByID("good1"), "a healthy device must be created")
							assert.NotNil(t, ad.ThingByID("good2"), "a healthy device after the broken one must be created")
							assert.Nil(t, ad.ThingByID("bad"), "the broken device must not be registered")
						}},
						Expectations: []*suite.Expectation{
							expectInclusion("good1").ExactlyOnce(),
							expectInclusion("good2").ExactlyOnce(),
						},
					},
					{
						Name:    "the broken device is retried rather than persisted as a ghost",
						Timeout: 500 * time.Millisecond,
						InitCallbacks: []suite.Callback{func(t *testing.T) {
							t.Helper()

							// A leftover state record would make EnsureThings treat "bad" as
							// already present and silently succeed.
							assert.ErrorIs(t, ad.EnsureThings(all), errFactory)
						}},
						Expectations: []*suite.Expectation{
							expectAnyExclusion().Never(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func expectAnyInclusion() *suite.Expectation {
	return suite.NewExpectation().
		ExpectTopic(testAdapterEvtTopic).
		ExpectType(adapter.EvtThingInclusionReport).
		ExpectService(testAdapterName)
}

func expectAnyExclusion() *suite.Expectation {
	return suite.NewExpectation().
		ExpectTopic(testAdapterEvtTopic).
		ExpectType(adapter.EvtThingExclusionReport).
		ExpectService(testAdapterName)
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
