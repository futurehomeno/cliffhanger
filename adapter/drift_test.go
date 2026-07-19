package adapter_test

import (
	"errors"
	"slices"
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

const testDriftAddress = "5"

// driftInfo drives the test factory: Groups feed the inclusion-report topology, Fail forces a
// factory error so the build-before-destroy guard can be exercised.
type driftInfo struct {
	Groups []string `json:"groups"`
	Fail   bool     `json:"fail"`
}

func TestRebuildChangedThings(t *testing.T) { //nolint:paralleltest
	var ad adapter.Adapter

	seed := func(info driftInfo) adapter.ThingSeeds {
		return adapter.ThingSeeds{{ID: "drift", CustomAddress: testDriftAddress, Info: info}}
	}

	setup := suite.BaseSetup(func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		factory := adapterhelper.FactoryHelper(func(_ adapter.Adapter, publisher adapter.Publisher, ts adapter.ThingState) (adapter.Thing, error) {
			var info driftInfo
			if err := ts.Info(&info); err != nil {
				return nil, err
			}

			if info.Fail {
				return nil, errors.New("factory boom")
			}

			cfg := &adapter.ThingConfig{
				InclusionReport: &fimptype.ThingInclusionReport{Address: ts.Address(), Groups: info.Groups},
				Connector:       mockedadapter.NewDefaultConnector(t),
			}

			return adapter.NewThing(publisher, ts, cfg), nil
		})

		ad = adapterhelper.PrepareSeededAdapter(t, testAdapterWorkDir, mqtt, factory, seed(driftInfo{Groups: []string{"g1"}}))

		return adapter.RouteAdapter(ad), nil, nil
	})

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "rebuild only when the service topology changes",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    setup,
				Nodes: []*suite.Node{
					{
						Name:    "a capability change rebuilds the thing",
						Timeout: 500 * time.Millisecond,
						InitCallbacks: []suite.Callback{func(t *testing.T) {
							t.Helper()
							assert.NoError(t, ad.RebuildChangedThings(seed(driftInfo{Groups: []string{"g1", "g2"}})))
						}},
						Expectations: []*suite.Expectation{
							expectExclusion(testDriftAddress).ExactlyOnce(),
							expectInclusionWithGroup(testDriftAddress, "g2").AtLeastOnce(),
						},
					},
					{
						Name:    "an unchanged seed does not rebuild",
						Timeout: 300 * time.Millisecond,
						InitCallbacks: []suite.Callback{func(t *testing.T) {
							t.Helper()
							assert.NoError(t, ad.RebuildChangedThings(seed(driftInfo{Groups: []string{"g1", "g2"}})))
						}},
						Expectations: []*suite.Expectation{
							expectExclusion(testDriftAddress).Never(),
						},
					},
					{
						Name:    "a factory error aborts before destroying the live thing",
						Timeout: 300 * time.Millisecond,
						InitCallbacks: []suite.Callback{func(t *testing.T) {
							t.Helper()
							assert.Error(t, ad.RebuildChangedThings(seed(driftInfo{Fail: true})),
								"a failed prospective build must surface as an error")
							assert.NotNil(t, ad.ThingByID("drift"), "the live thing must survive a failed rebuild")
						}},
						Expectations: []*suite.Expectation{
							expectExclusion(testDriftAddress).Never(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func expectExclusion(address string) *suite.Expectation {
	return suite.NewExpectation().
		ExpectTopic(testAdapterEvtTopic).
		ExpectType(adapter.EvtThingExclusionReport).
		ExpectService(testAdapterName).
		Expect(router.MessageVoterFn(func(m *fimpgo.Message) bool {
			var report fimptype.ThingExclusionReport

			if err := m.Payload.GetObjectValue(&report); err != nil {
				return false
			}

			return report.Address == address
		}))
}

func expectInclusionWithGroup(address, group string) *suite.Expectation {
	return suite.NewExpectation().
		ExpectTopic(testAdapterEvtTopic).
		ExpectType(adapter.EvtThingInclusionReport).
		ExpectService(testAdapterName).
		Expect(router.MessageVoterFn(func(m *fimpgo.Message) bool {
			var report fimptype.ThingInclusionReport

			if err := m.Payload.GetObjectValue(&report); err != nil {
				return false
			}

			return report.Address == address && slices.Contains(report.Groups, group)
		}))
}
