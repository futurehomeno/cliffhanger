package adapter

import (
	"errors"
	"fmt"
	"time"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/prime"
)

// staleNodeTimeout bounds the Vinculum request, which runs inline in a device sync.
const staleNodeTimeout = 10 * time.Second

// Option configures an adapter at construction time.
type Option func(*adapter)

// WithStaleNodeExclusion overrides the stale node sweep, which is enabled by default. The
// predicate is evaluated when the sweep is due, so it can be backed by a configuration setting.
func WithStaleNodeExclusion(enabled func() bool) Option {
	return func(a *adapter) { a.staleNodes = enabled }
}

// excludeStaleNodesOnce sweeps stale hub nodes at most once per process, and is called from
// SyncThings once a device list was successfully fetched and applied - the only moment the
// adapter's things are known to match the service's, so that a node the sync is about to
// recreate can never be excluded first. Failures are logged: a sweep must not break a sync.
func (a *adapter) excludeStaleNodesOnce() {
	// Before initialization existing records are not yet live, so every hub node would look stale
	// and the whole fleet would be excluded. A racing device sync can still report success with an
	// empty or partial live map (heal is skipped; destroy is deferred; new seeds may already be
	// registered). Checked outside the once so the sweep is only deferred, not lost: the next sync
	// after initialization still gets it.
	if !a.IsInitialized() {
		return
	}

	a.staleNodesOnce.Do(func() {
		if a.mqtt == nil || a.staleNodes == nil || !a.staleNodes() {
			return
		}

		// Asking the hub is a round trip over MQTT: a sweep must never hold up the sync it
		// is triggered from, nor block it for the full timeout when nothing answers.
		go a.excludeStaleNodes()
	})
}

func (a *adapter) excludeStaleNodes() {
	excluded, err := ExcludeStaleNodes(a, prime.NewClient(fimpgo.NewSyncClient(a.mqtt), a.name, staleNodeTimeout))
	if err != nil {
		log.Errorf("[adapter] Exclude stale hub nodes. err: %v", err)
	}

	for _, address := range excluded {
		log.Infof("[adapter] Excluded stale hub node %s", address)
	}
}

// ExcludeStaleNodes asks the hub which nodes it attributes to this adapter and announces an
// exclusion for every one the adapter owns no thing for, clearing nodes left behind by a state
// store that was lost or restored from an older backup. Returns the addresses it excluded.
//
// It must only be called when the adapter's things are known to reflect a successful device
// fetch, or a node the adapter is about to register would be excluded moments before.
// Best-effort per address: failures are joined and the addresses that did get excluded are
// still returned.
func ExcludeStaleNodes(a Adapter, client prime.Client) ([]string, error) {
	devices, err := client.GetDevices()
	if err != nil {
		return nil, fmt.Errorf("adapter: fetch hub devices: %w", err)
	}

	var (
		excluded []string
		errs     []error
	)

	for _, address := range staleAddresses(a, devices) {
		if err := a.DestroyThingByAddress(address); err != nil {
			errs = append(errs, fmt.Errorf("adapter: exclude stale node %s: %w", address, err))

			continue
		}

		excluded = append(excluded, address)
	}

	return excluded, errors.Join(errs...)
}

// staleAddresses returns the addresses of hub nodes attributed to this adapter that the adapter
// owns no thing for.
func staleAddresses(a Adapter, devices prime.Devices) []string {
	// Doubles as the dedupe set: several hub devices can share a single thing address.
	seen := make(map[string]struct{})

	for _, t := range a.Things() {
		seen[t.Address()] = struct{}{}
	}

	var stale []string

	for _, device := range devices {
		address := device.FIMP.Address
		if address == "" || !belongsToAdapter(a, device) {
			continue
		}

		if _, done := seen[address]; done {
			continue
		}

		seen[address] = struct{}{}

		stale = append(stale, address)
	}

	return stale
}

// belongsToAdapter reports whether the hub attributes the device to this adapter. The resource
// name and instance are taken from a service topic rather than from device.FIMP.Adapter, which
// carries the service name - the two differ for technologies such as zwave.
func belongsToAdapter(a Adapter, device *prime.Device) bool {
	if device.FIMP.AdapterAddress != "" && device.FIMP.AdapterAddress != a.Address() {
		return false
	}

	for _, service := range device.Services {
		address, err := fimpgo.NewAddressFromString(service.Addr)
		if err != nil || address.ResourceName == "" {
			continue
		}

		return address.ResourceName == a.Name() &&
			(address.ResourceAddress == "" || address.ResourceAddress == a.Address())
	}

	return false
}
