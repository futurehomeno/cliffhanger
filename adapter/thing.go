package adapter

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"strings"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

// serviceAddressPrefix is what every service specification address starts with.
const serviceAddressPrefix = "/rt:dev/rn:"

type ThingRegistry interface {
	Things() []Thing
	ThingByAddress(address string) Thing
	ThingByTopic(topic string) Thing
}

type ThingFactory interface {
	// Create creates an instance of a thing using provided state.
	//
	// RebuildChangedThings may call Create speculatively, on a prospective thing that is
	// discarded without ever being registered if its topology turns out unchanged. Create
	// must therefore not perform work whose effect is meant to persist beyond the returned
	// Thing - e.g. registering the thing with an external manager, or mutating storage keyed
	// by its topic - since that side effect is not undone when the prospective build is
	// thrown away.
	//
	// Create is always called with the adapter's write lock held, on a non-reentrant mutex.
	// The adapter is passed so that it can be handed to the thing being built; calling back into
	// it - ThingByAddress, Things, ServiceByTopic and so on - deadlocks the whole adapter.
	Create(adapter Adapter, publisher Publisher, thingState ThingState) (Thing, error)
}

type ThingSeeds []*ThingSeed

func (s ThingSeeds) Contains(id string) bool {
	for _, seed := range s {
		if seed.ID == id {
			return true
		}
	}

	return false
}

func (s ThingSeeds) Without(id string) ThingSeeds {
	var seeds ThingSeeds

	for _, seed := range s {
		if seed.ID == id {
			continue
		}

		seeds = append(seeds, seed)
	}

	return seeds
}

type ThingSeed struct {
	ID            string
	Info          any
	CustomAddress string
}

type ThingConfig struct {
	Connector                     Connector
	InclusionReport               *fimptype.ThingInclusionReport
	ConnectivityReportingStrategy cache.ReportingStrategy
}

type Thing interface {
	// Update updates the thing by applying the list of ThingUpdate functions. Sending report is optional.
	// Returns error if failed to send report.
	Update(...ThingUpdate) error
	// Address returns address of the thing.
	Address() string
	// Services returns all services from the thing that match the provided name. If empty all services are returned.
	Services(name fimptype.ServiceNameT) []Service // map[topic][]Service
	// ServiceByTopic returns a service based on the topic on which is supposed to be listening for commands.
	ServiceByTopic(topic string) Service
	InclusionReport() *fimptype.ThingInclusionReport
	// If force is true, report is sent even if it did not change from previously sent one.
	SendInclusionReport(force bool) (bool, error)
	ConnectivityReport() *ConnectivityReport
	// If force is true, report is sent even if it did not change from previously sent one.
	SendConnectivityReport(force bool) (bool, error)
	SendPingReport() error
	// If the thing is already connected, this method does nothing.
	Connect()
	// Disconnect disconnects the thing. If the thing is already disconnected, this method does nothing.
	Disconnect()
}

func NewThing(
	publisher Publisher,
	state ThingState,
	cfg *ThingConfig,
	services ...Service,
) Thing {
	if cfg.ConnectivityReportingStrategy == nil {
		cfg.ConnectivityReportingStrategy = cache.ReportAtLeastEvery(time.Hour)
	}

	cfg.InclusionReport.Services = nil

	servicesIndex := make(map[string]Service)

	for _, s := range services {
		servicesIndex[s.Topic()] = s
		cfg.InclusionReport.Services = append(cfg.InclusionReport.Services, *s.Specification())
	}

	return &thing{
		publisher:                     publisher,
		state:                         state,
		connector:                     cfg.Connector,
		reportingCache:                cache.NewReportingCache(),
		connectivityReportingStrategy: cfg.ConnectivityReportingStrategy,
		inclusionReport:               cfg.InclusionReport,
		services:                      servicesIndex,
		lock:                          &sync.RWMutex{},
	}
}

type thing struct {
	publisher                     Publisher
	state                         ThingState
	connector                     Connector
	reportingCache                cache.ReportingCache
	connectivityReportingStrategy cache.ReportingStrategy
	inclusionReport               *fimptype.ThingInclusionReport
	services                      map[string]Service
	lock                          *sync.RWMutex
}

func (t *thing) Address() string {
	return t.inclusionReport.Address
}

func (t *thing) Services(name fimptype.ServiceNameT) []Service {
	t.lock.RLock()
	defer t.lock.RUnlock()

	var services []Service

	for _, s := range t.services {
		if name != "" && s.Name() != name {
			continue
		}

		services = append(services, s)
	}

	return services
}

func (t *thing) ServiceByTopic(topic string) Service {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.services[serviceTopicKey(topic)]
}

// serviceTopicKey reduces a topic to the service address the services index is keyed by. Service
// addresses always start with /rt:dev/rn:, while an inbound message topic carries a prefix in front
// of it, so a lookup only needs to cut everything before that marker. A topic without the marker is
// returned as it is: it can only miss, which is what scanning for a matching suffix did too.
func serviceTopicKey(topic string) string {
	if i := strings.Index(topic, serviceAddressPrefix); i >= 0 {
		return topic[i:]
	}

	return topic
}

func (t *thing) InclusionReport() *fimptype.ThingInclusionReport {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.inclusionReport
}

// If force is true, report is sent even if it did not change from previously sent one.
func (t *thing) SendInclusionReport(force bool) (bool, error) {
	report := t.InclusionReport()

	t.lock.Lock()
	defer t.lock.Unlock()

	data, err := json.Marshal(report)
	if err != nil {
		return false, fmt.Errorf("thing: failed to marshal inclusion report: %w", err)
	}

	checksum := crc32.ChecksumIEEE(data)

	if !force && checksum == t.state.InclusionChecksum() {
		return false, nil
	}

	message := fimpgo.NewObjectMessage(
		EvtThingInclusionReport,
		"",
		t.inclusionReport,
		nil,
		nil,
		nil,
	)

	err = t.publisher.PublishAdapterMessage(message)
	if err != nil {
		return false, fmt.Errorf("thing: failed to send inclusion report: %w", err)
	}

	t.publisher.PublishThingEvent(NewInclusionReportSentEvent(t.Address(), *report))

	err = t.state.SetInclusionChecksum(checksum)
	if err != nil {
		return false, fmt.Errorf("thing: failed to set inclusion checksum: %w", err)
	}

	return true, nil
}

func (t *thing) ConnectivityReport() *ConnectivityReport {
	t.lock.Lock()
	defer t.lock.Unlock()

	connectivityDetails := t.connector.Connectivity()

	report := &ConnectivityReport{
		Address:             t.Address(),
		Hash:                t.inclusionReport.ProductHash,
		Alias:               t.inclusionReport.ProductName,
		PowerSource:         t.inclusionReport.PowerSource,
		WakeupInterval:      t.inclusionReport.WakeUpInterval,
		CommTechnology:      t.inclusionReport.CommTechnology,
		ConnectivityDetails: connectivityDetails,
	}

	report.sanitize()

	return report
}

// If force is true, report is sent even if it did not change from previously sent one.
func (t *thing) SendConnectivityReport(force bool) (bool, error) {
	report := t.ConnectivityReport()

	t.lock.Lock()
	defer t.lock.Unlock()

	t.publisher.PublishThingEvent(newConnectivityEvent(t, report.ConnectivityDetails))

	if !force && !t.reportingCache.ReportRequired(t.connectivityReportingStrategy, EvtNetworkNodeReport, "", report) {
		return false, nil
	}

	message := fimpgo.NewObjectMessage(
		EvtNetworkNodeReport,
		"",
		report,
		nil,
		nil,
		nil,
	)

	err := t.publisher.PublishThingMessage(t, message)
	if err != nil {
		return false, fmt.Errorf("thing: failed to send node report: %w", err)
	}

	t.reportingCache.Reported(EvtNetworkNodeReport, "", report)

	return true, nil
}

func (t *thing) SendPingReport() error {
	t.lock.RLock()
	defer t.lock.RUnlock()

	ts := time.Now()

	pingDetails := t.connector.Ping()

	delay := int(time.Since(ts).Truncate(time.Millisecond) / time.Millisecond)

	report := &PingReport{
		Address:     t.Address(),
		Delay:       delay,
		PingDetails: pingDetails,
	}

	message := fimpgo.NewObjectMessage(
		EvtPingReport,
		"",
		report,
		nil,
		nil,
		nil,
	)

	err := t.publisher.PublishThingMessage(t, message)
	if err != nil {
		return fmt.Errorf("thing: failed to send ping report: %w", err)
	}

	return nil
}

// If the thing is already connected, this method does nothing.
func (t *thing) Connect() {
	c, ok := t.connector.(ControllableConnector)
	if !ok {
		return
	}

	c.Connect(t)
}

func (t *thing) Disconnect() {
	c, ok := t.connector.(ControllableConnector)
	if !ok {
		return
	}

	c.Disconnect(t)
}

type ThingUpdate func(*thing)

func (t *thing) Update(options ...ThingUpdate) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	for _, o := range options {
		o.Apply(t)
	}

	return nil
}

func (o ThingUpdate) Apply(t *thing) {
	o(t)
}

func ThingUpdateAddService(s Service) ThingUpdate {
	return func(t *thing) {
		if t.services[s.Topic()] == nil {
			t.services[s.Topic()] = s
			t.inclusionReport.Services = append(t.inclusionReport.Services, *s.Specification())
		}
	}
}

func ThingUpdateRemoveService(s Service) ThingUpdate {
	return func(t *thing) {
		delete(t.services, s.Topic())

		newServices := make([]fimptype.Service, 0)

		for _, srv := range t.inclusionReport.Services {
			if s.Topic() != srv.Address {
				newServices = append(newServices, srv)
			}
		}

		t.inclusionReport.Services = newServices
	}
}
