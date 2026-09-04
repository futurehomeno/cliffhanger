package adapter_test

import (
	"errors"
	"testing"

	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/prime"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedprime "github.com/futurehomeno/cliffhanger/test/mocks/prime"
)

var errHubDevices = errors.New("hub unreachable")

// hubDevice builds a device as the hub reports it, addressed under the provided resource name.
func hubDevice(resourceName, address string) *prime.Device {
	return hubDeviceForInstance(resourceName, "1", address)
}

// hubDeviceForInstance builds a device whose service topic names the given adapter instance,
// with device.FIMP.AdapterAddress left empty - the fallback path used for technologies such as
// zwave that don't report the instance on the device itself.
func hubDeviceForInstance(resourceName, instance, address string) *prime.Device {
	return &prime.Device{
		FIMP: prime.DeviceFIMP{Address: address},
		Services: map[fimptype.ServiceNameT]*prime.Service{
			"thermostat": {Addr: "pt:j1/mt:evt/rt:dev/rn:" + resourceName + "/ad:" + instance + "/sv:thermostat/ad:" + address},
		},
	}
}

// staleAdapter returns an adapter mock owning things at the provided addresses.
func staleAdapter(t *testing.T, addresses ...string) *mockedadapter.Adapter {
	t.Helper()

	a := mockedadapter.NewAdapter(t)
	a.On("Name").Return(fimptype.ResourceNameT("mill")).Maybe()
	a.On("Address").Return("1").Maybe()

	things := make([]adapter.Thing, 0, len(addresses))

	for _, address := range addresses {
		thing := mockedadapter.NewThing(t)
		thing.On("Address").Return(address).Maybe()

		things = append(things, thing)
	}

	a.On("Things").Return(things).Maybe()

	return a
}

func TestExcludeStaleNodes_ExcludesNodesTheAdapterDoesNotOwn(t *testing.T) {
	t.Parallel()

	a := staleAdapter(t, "1")
	a.On("DestroyThingByAddress", "2").Return(nil)

	c := mockedprime.NewClient(t)
	c.On("GetDevices").Return(prime.Devices{
		hubDevice("mill", "1"),
		hubDevice("mill", "2"),
		// A second hub device sharing the stale thing must not be excluded twice.
		hubDevice("mill", "2"),
		hubDevice("zigbee", "3"),
	}, nil)

	excluded, err := adapter.ExcludeStaleNodes(a, c)

	assert.NoError(t, err)
	assert.Equal(t, []string{"2"}, excluded)
	a.AssertNumberOfCalls(t, "DestroyThingByAddress", 1)
}

func TestExcludeStaleNodes_FetchFailureExcludesNothing(t *testing.T) {
	t.Parallel()

	a := mockedadapter.NewAdapter(t)

	c := mockedprime.NewClient(t)
	c.On("GetDevices").Return(nil, errHubDevices)

	excluded, err := adapter.ExcludeStaleNodes(a, c)

	assert.ErrorIs(t, err, errHubDevices)
	assert.ErrorIs(t, err, adapter.ErrStaleNodeFetch, "the sweep claim is released on this error alone")
	assert.Empty(t, excluded)
	a.AssertNotCalled(t, "DestroyThingByAddress", mock.Anything)
}

func TestExcludeStaleNodes_ReportsFailuresAndKeepsGoing(t *testing.T) {
	t.Parallel()

	a := staleAdapter(t)
	a.On("DestroyThingByAddress", "1").Return(errors.New("publish failed"))
	a.On("DestroyThingByAddress", "2").Return(nil)

	c := mockedprime.NewClient(t)
	c.On("GetDevices").Return(prime.Devices{hubDevice("mill", "1"), hubDevice("mill", "2")}, nil)

	excluded, err := adapter.ExcludeStaleNodes(a, c)

	assert.Error(t, err)
	assert.NotErrorIs(t, err, adapter.ErrStaleNodeFetch,
		"a partial failure must keep the claim, or every later sync re-excludes the nodes that worked")
	assert.Equal(t, []string{"2"}, excluded)
}

func TestExcludeStaleNodes_IgnoresNodesOfAnotherAdapterInstance(t *testing.T) {
	t.Parallel()

	a := staleAdapter(t)

	device := hubDevice("mill", "2")
	device.FIMP.AdapterAddress = "2"

	c := mockedprime.NewClient(t)
	c.On("GetDevices").Return(prime.Devices{device}, nil)

	excluded, err := adapter.ExcludeStaleNodes(a, c)

	assert.NoError(t, err)
	assert.Empty(t, excluded)
	a.AssertNotCalled(t, "DestroyThingByAddress", mock.Anything)
}

func TestExcludeStaleNodes_IgnoresNodesOfAnotherAdapterInstanceViaServiceTopic(t *testing.T) {
	t.Parallel()

	a := staleAdapter(t)

	// AdapterAddress is empty, as it is for zwave devices; only the service topic names the
	// instance, and it belongs to instance "2" while the adapter under test is instance "1".
	device := hubDeviceForInstance("mill", "2", "2")

	c := mockedprime.NewClient(t)
	c.On("GetDevices").Return(prime.Devices{device}, nil)

	excluded, err := adapter.ExcludeStaleNodes(a, c)

	assert.NoError(t, err)
	assert.Empty(t, excluded)
	a.AssertNotCalled(t, "DestroyThingByAddress", mock.Anything)
}
