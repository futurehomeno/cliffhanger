package adapter_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/selection"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

// deletableDevices mirrors setupAdapterWithDeletableThings: the ID differs from the address, as
// it does for every adapter that lets cliffhanger assign addresses, so a handler that mistook one
// for the other would be caught.
var deletableDevices = []device{{ID: "B", Name: testThingAddressB}, {ID: "C", Name: testThingAddressC}}

// seedDeletable maps a device to a seed pinned at the address carried in Name.
func seedDeletable(d device) *adapter.ThingSeed {
	return &adapter.ThingSeed{ID: d.ID, CustomAddress: d.Name}
}

// selection.Store must satisfy the interface the delete route accepts; its doc comment says so.
var _ adapter.SelectionRemover = (*selection.Store)(nil)

// testSelection stands in for an application configuration. It locks around the selection because
// NewStore requires it: the delete handler writes from the router goroutine while the test reads.
type testSelection struct {
	lock sync.RWMutex
	sel  selection.Selection
}

// newTestSelection returns a store over an in-memory selection along with the backing fixture, for
// inspecting what the handler wrote.
func newTestSelection(ids ...string) (*selection.Store, *testSelection) {
	s := &testSelection{sel: selection.Selection(ids)}

	return selection.NewStore(s.get, s.set), s
}

func (s *testSelection) get() selection.Selection {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.sel.Clone()
}

func (s *testSelection) set(next selection.Selection) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.sel = next

	return nil
}

// failingRemover stands in for a selection whose configuration write fails, e.g. a full disk.
type failingRemover struct{}

func (failingRemover) Remove(string) error { return errors.New("selection write failed") }

func setupAdapterWithDeletableThings(
	ad *adapter.Adapter,
	remover adapter.SelectionRemover,
	locker router.MessageHandlerLocker,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		factory := adapterhelper.FactoryHelper(func(_ adapter.Adapter, publisher adapter.Publisher, ts adapter.ThingState) (adapter.Thing, error) {
			cfg := &adapter.ThingConfig{
				InclusionReport: &fimptype.ThingInclusionReport{Address: ts.Address()},
				Connector:       mockedadapter.NewDefaultConnector(t),
			}

			return adapter.NewThing(publisher, ts, cfg), nil
		})

		seeds := adapter.SeedsFromSelection(deletableDevices, nil, seedDeletable)

		*ad = adapterhelper.PrepareSeededAdapter(t, testAdapterWorkDir, mqtt, factory, seeds)

		return adapter.RouteAdapter(*ad, adapter.WithSelection(remover, locker)), nil, nil
	}
}

func deleteThingCommand(address string) *fimpgo.Message {
	return suite.StringMapMessage(
		testAdapterCmdTopic,
		adapter.CmdThingDelete,
		testAdapterName,
		map[string]string{"address": address},
	)
}

func TestRouteAdapter_ThingDelete(t *testing.T) { //nolint:paralleltest
	var ad adapter.Adapter

	store, sel := newTestSelection("B", "C")

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "cmd.thing.delete destroys the thing, announces it and deselects the device",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    setupAdapterWithDeletableThings(&ad, store, router.NewMessageHandlerLocker()),
				Nodes: []*suite.Node{
					{
						Name:    "deleting thing B excludes it",
						Timeout: 500 * time.Millisecond,
						Command: deleteThingCommand(testThingAddressB),
						Expectations: []*suite.Expectation{
							expectExclusion(testThingAddressB).ExactlyOnce(),
							suite.ExpectError(testAdapterEvtTopic, testAdapterName).Never(),
						},
					},
					{
						Name:    "the device is deselected and a following sync does not bring it back",
						Timeout: 500 * time.Millisecond,
						InitCallbacks: []suite.Callback{func(t *testing.T) {
							t.Helper()

							assert.Nil(t, ad.ThingByAddress(testThingAddressB), "thing B must be destroyed")
							// "B", not "2": the ID is resolved before the destroy drops the
							// record that maps the address to it.
							assert.Equal(t, selection.Selection{"C"}, sel.get())

							// The third party still lists both devices; only the selection
							// decides, which is what makes the deletion stick.
							_, err := adapter.SyncThings(ad, fetchOK(deletableDevices), store.Get(), seedDeletable)

							assert.NoError(t, err)
							assert.Nil(t, ad.ThingByAddress(testThingAddressB), "the deleted device must stay gone")
							assert.NotNil(t, ad.ThingByAddress(testThingAddressC), "the kept device must survive")
						}},
						Expectations: []*suite.Expectation{
							expectInclusion(testThingAddressB).Never(),
						},
					},
					{
						Name:    "re-selecting the device recreates it and announces it again",
						Timeout: 500 * time.Millisecond,
						InitCallbacks: []suite.Callback{func(t *testing.T) {
							t.Helper()

							assert.NoError(t, store.Set(selection.Selection{"B", "C"}))

							_, err := adapter.SyncThings(ad, fetchOK(deletableDevices), store.Get(), seedDeletable)

							assert.NoError(t, err)
							assert.NotNil(t, ad.ThingByAddress(testThingAddressB), "the re-selected device must be back")
						}},
						Expectations: []*suite.Expectation{
							expectInclusion(testThingAddressB).ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestRouteAdapter_ThingDeleteWithoutSelection(t *testing.T) { //nolint:paralleltest
	// Without a selection remover the handler keeps its original behaviour: destroy and
	// announce, leaving the application's configuration alone.
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "cmd.thing.delete destroys the thing when no selection is wired",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    setupAdapterWithTwoThings(),
				Nodes: []*suite.Node{
					{
						Name:    "deleting thing B excludes it",
						Timeout: 500 * time.Millisecond,
						Command: deleteThingCommand(testThingAddressB),
						Expectations: []*suite.Expectation{
							expectExclusion(testThingAddressB).ExactlyOnce(),
							suite.ExpectError(testAdapterEvtTopic, testAdapterName).Never(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestRouteAdapter_ThingDeleteRespectsExternalLock(t *testing.T) { //nolint:paralleltest
	var ad adapter.Adapter

	store, sel := newTestSelection("B", "C")
	// The same locker app.RouteApp holds while it rewrites the configuration. A delete that slipped
	// past it would read a selection the configuration handler is midway through replacing, and the
	// deselect would be lost.
	locker := router.NewMessageHandlerLocker()

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "cmd.thing.delete is skipped while the configuration lock is held",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    setupAdapterWithDeletableThings(&ad, store, locker),
				Nodes: []*suite.Node{
					{
						Name:    "a delete arriving under the held lock changes nothing",
						Timeout: 500 * time.Millisecond,
						InitCallbacks: []suite.Callback{func(t *testing.T) {
							t.Helper()

							assert.True(t, locker.Lock(), "the test must own the lock the handler contends for")
						}},
						Command: deleteThingCommand(testThingAddressB),
						Expectations: []*suite.Expectation{
							suite.ExpectError(testAdapterEvtTopic, testAdapterName).ExactlyOnce(),
							expectAnyExclusion().Never(),
						},
					},
					{
						Name:    "releasing the lock lets the same delete through",
						Timeout: 500 * time.Millisecond,
						InitCallbacks: []suite.Callback{func(t *testing.T) {
							t.Helper()

							assert.NotNil(t, ad.ThingByAddress(testThingAddressB), "the blocked delete must not have destroyed it")
							assert.Equal(t, selection.Selection{"B", "C"}, sel.get(), "the blocked delete must not have deselected it")

							locker.Unlock()
						}},
						Command: deleteThingCommand(testThingAddressB),
						Expectations: []*suite.Expectation{
							expectExclusion(testThingAddressB).ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestRouteAdapter_WithSelectionRejectsNilArguments(t *testing.T) {
	t.Parallel()

	// A nil locker leaves the handler unlocked, which is exactly the race the option exists to
	// close, so it must not be silently accepted.
	assert.Panics(t, func() { adapter.WithSelection(nil, router.NewMessageHandlerLocker()) })
	assert.Panics(t, func() { adapter.WithSelection(&selection.Store{}, nil) })
}

func TestRouteAdapter_ThingDeleteSelectionWriteFails(t *testing.T) { //nolint:paralleltest
	var ad adapter.Adapter

	// The deselect runs before the destroy precisely so this case is recoverable: were the order
	// reversed, the thing would be gone while the device stayed selected and the next sync would
	// resurrect the device the user just deleted.
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "a failing deselect leaves the thing intact instead of resurrecting it",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    setupAdapterWithDeletableThings(&ad, failingRemover{}, router.NewMessageHandlerLocker()),
				Nodes: []*suite.Node{
					{
						Name:    "the delete reports an error and announces no exclusion",
						Timeout: 500 * time.Millisecond,
						Command: deleteThingCommand(testThingAddressB),
						Expectations: []*suite.Expectation{
							suite.ExpectError(testAdapterEvtTopic, testAdapterName).ExactlyOnce(),
							expectAnyExclusion().Never(),
						},
					},
					{
						Name:    "the thing survives, so retrying the delete is all it takes",
						Timeout: 500 * time.Millisecond,
						InitCallbacks: []suite.Callback{func(t *testing.T) {
							t.Helper()

							assert.NotNil(t, ad.ThingByAddress(testThingAddressB), "thing B must survive a failed deselect")
						}},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestRouteAdapter_ThingDeleteErrorPaths(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "cmd.thing.delete error paths never announce an exclusion",
				TearDown: adapterhelper.TearDownAdapter(testAdapterWorkDir),
				Setup:    setupAdapterWithTwoThings(),
				Nodes: []*suite.Node{
					{
						Name:    "an empty address responds with an error",
						Timeout: 500 * time.Millisecond,
						Command: deleteThingCommand(""),
						Expectations: []*suite.Expectation{
							suite.ExpectError(testAdapterEvtTopic, testAdapterName).ExactlyOnce(),
							expectAnyExclusion().Never(),
						},
					},
					{
						Name:    "a missing address key responds with an error",
						Timeout: 500 * time.Millisecond,
						Command: suite.StringMapMessage(
							testAdapterCmdTopic,
							adapter.CmdThingDelete,
							testAdapterName,
							map[string]string{},
						),
						Expectations: []*suite.Expectation{
							suite.ExpectError(testAdapterEvtTopic, testAdapterName).ExactlyOnce(),
							expectAnyExclusion().Never(),
						},
					},
					{
						Name:    "a payload that is not a string map responds with an error",
						Timeout: 500 * time.Millisecond,
						Command: suite.StringMessage(
							testAdapterCmdTopic,
							adapter.CmdThingDelete,
							testAdapterName,
							testThingAddressB,
						),
						Expectations: []*suite.Expectation{
							suite.ExpectError(testAdapterEvtTopic, testAdapterName).ExactlyOnce(),
							expectAnyExclusion().Never(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}
