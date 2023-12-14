package adapter

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"strings"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

// ThingFactory is an interface representing a thing factory service which is used by a stateful adapter.
type ThingFactory interface {
	// Create creates an instance of a thing using provided state.
	Create(adapter Adapter, publisher Publisher, thingState ThingState) (Thing, error)
}

// ThingSeeds is a set of thing seeds to be used for creating things.
type ThingSeeds []*ThingSeed

// Contains returns true if the set contains a thing with the provided ID.
func (s ThingSeeds) Contains(id string) bool {
	for _, seed := range s {
		if seed.ID == id {
			return true
		}
	}

	return false
}

// Without returns a new set without a thing with the provided ID.
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

// ThingSeed represents a thing seed to be used for creating thing.
type ThingSeed struct {
	ID            string
	Info          interface{}
	CustomAddress string
}

// ThingConfig represents a thing configuration.
type ThingConfig struct {
	Connector                     Connector
	InclusionReport               *fimptype.ThingInclusionReport
	ConnectivityReportingStrategy cache.ReportingStrategy
}

type ThingUpdate func(*thing)

// Thing is an interface representing FIMP thing.
type Thing interface {
	// Update updates the thing by applying the list of ThingUpdate functions. Sending report is optional.
	// Returns error if failed to send report.
	Update(bool, ...ThingUpdate) error
	// Address returns address of the thing.
	Address() string
	// Services returns all services from the thing that match the provided name. If empty all services are returned.
	Services(name string) []Service
	// ServiceByTopic returns a service based on the topic on which is supposed to be listening for commands.
	ServiceByTopic(topic string) Service
	// InclusionReport returns an inclusion report of the thing.
	InclusionReport() *fimptype.ThingInclusionReport
	// SendInclusionReport sends inclusion report of the thing.
	// If force is true, report is sent even if it did not change from previously sent one.
	SendInclusionReport(force bool) (bool, error)
	// ConnectivityReport returns a connectivity report of the thing.
	ConnectivityReport() *ConnectivityReport
	// SendConnectivityReport sends connectivity report of the thing.
	// If force is true, report is sent even if it did not change from previously sent one.
	SendConnectivityReport(force bool) (bool, error)
	// SendPingReport sends ping report of the thing.
	SendPingReport() error
	// Connect connects the thing. If the thing is already connected, this method does nothing.
	Connect()
	// Disconnect disconnects the thing. If the thing is already disconnected, this method does nothing.
	Disconnect()
}

// NewThing creates new instance of a FIMP thing.
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
	}
}

// thing is a private implementation of a FIMP thing.
type thing struct {
	publisher                     Publisher
	state                         ThingState
	connector                     Connector
	reportingCache                cache.ReportingCache
	connectivityReportingStrategy cache.ReportingStrategy
	inclusionReport               *fimptype.ThingInclusionReport
	services                      map[string]Service
}

// Address returns address of the thing.
func (t *thing) Address() string {
	return t.inclusionReport.Address
}

// Services returns all services from the thing that match the provided name. If empty all services are returned.
func (t *thing) Services(name string) []Service {
	var services []Service

	for _, s := range t.services {
		if name != "" && s.Name() != name {
			continue
		}

		services = append(services, s)
	}

	return services
}

// ServiceByTopic returns a service based on the topic on which it is supposed to be listening for commands.
func (t *thing) ServiceByTopic(topic string) Service {
	for serviceTopic, s := range t.services {
		if strings.HasSuffix(topic, serviceTopic) {
			return s
		}
	}

	return nil
}

// InclusionReport returns an inclusion report of the thing.
func (t *thing) InclusionReport() *fimptype.ThingInclusionReport {
	return t.inclusionReport
}

// SendInclusionReport sends inclusion report of the thing.
// If force is true, report is sent even if it did not change from previously sent one.
func (t *thing) SendInclusionReport(force bool) (bool, error) {
	report := t.InclusionReport()

	data, err := json.Marshal(report)
	if err != nil {
		return false, fmt.Errorf("thing: failed to marshal inclusion report: %w", err)
	}

	checksum := crc32.ChecksumIEEE(data)

	if !force && checksum == t.state.GetInclusionChecksum() {
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

	err = t.state.SetInclusionChecksum(checksum)
	if err != nil {
		return false, fmt.Errorf("thing: failed to set inclusion checksum: %w", err)
	}

	return true, nil
}

// ConnectivityReport returns a connectivity report of the thing.
func (t *thing) ConnectivityReport() *ConnectivityReport {
	report := &ConnectivityReport{
		Address:             t.Address(),
		Hash:                t.inclusionReport.ProductHash,
		Alias:               t.inclusionReport.ProductName,
		PowerSource:         t.inclusionReport.PowerSource,
		WakeupInterval:      t.inclusionReport.WakeUpInterval,
		CommTechnology:      t.inclusionReport.CommTechnology,
		ConnectivityDetails: t.connector.Connectivity(),
	}

	report.sanitize()

	return report
}

// SendConnectivityReport sends connectivity report of the thing.
// If force is true, report is sent even if it did not change from previously sent one.
func (t *thing) SendConnectivityReport(force bool) (bool, error) {
	report := t.ConnectivityReport()

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

// SendPingReport sends ping report of the thing.
func (t *thing) SendPingReport() error {
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

// Connect connects the thing. If the thing is already connected, this method does nothing.
func (t *thing) Connect() {
	c, ok := t.connector.(ControllableConnector)
	if !ok {
		return
	}

	c.Connect(t)
}

// Disconnect disconnects the thing. If the thing is already disconnected, this method does nothing.
func (t *thing) Disconnect() {
	c, ok := t.connector.(ControllableConnector)
	if !ok {
		return
	}

	c.Disconnect(t)
}

// Update applies provided ThingUpdate options to the thing and sends a report if requested.
func (t *thing) Update(report bool, options ...ThingUpdate) error {
	for _, o := range options {
		o.Apply(t)
	}

	if report {
		if _, err := t.SendInclusionReport(true); err != nil {
			return fmt.Errorf("failed to send inclusion report when updating thing: %w", err)
		}
	}

	return nil
}

func (o ThingUpdate) Apply(t *thing) {
	o(t)
}

func ThingUpdateAddService(s Service) ThingUpdate {
	return func(t *thing) {
		t.services[s.Topic()] = s
		t.inclusionReport.Services = append(t.inclusionReport.Services, *s.Specification())
	}
}

func ThingUpdateRemoveService(s Service) ThingUpdate {
	return func(t *thing) {
		delete(t.services, s.Topic())

		newServices := make([]fimptype.Service, 0, len(t.inclusionReport.Services))

		for _, srv := range t.inclusionReport.Services {
			if s.Name() != srv.Name {
				newServices = append(newServices, srv)
			}
		}

		t.inclusionReport.Services = newServices
	}
}
