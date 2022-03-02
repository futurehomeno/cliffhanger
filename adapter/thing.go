package adapter

import (
	"strings"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
)

// ThingFactory is an interface representing a thing factory service which is used by a stateful adapter.
type ThingFactory interface {
	// Create creates an instance of a thing using provided state.
	Create(mqtt *fimpgo.MqttTransport, adapter ExtendedAdapter, thingState ThingState) (Thing, error)
}

// Thing is an interface representing FIMP thing.
type Thing interface {
	// InclusionReport returns an inclusion report of the thing.
	InclusionReport() *fimptype.ThingInclusionReport
	// Address returns address of the thing.
	Address() string
	// Services returns all services from the thing that match the provided name. If empty all services are returned.
	Services(name string) []Service
	// ServiceByTopic returns a service based on the topic on which is is supposed to be listening for commands.
	ServiceByTopic(topic string) Service
}

// NewThing creates new instance of a FIMP thing.
func NewThing(thingInclusionReport *fimptype.ThingInclusionReport, services ...Service) Thing {
	servicesIndex := make(map[string]Service)
	thingInclusionReport.Services = nil

	for _, s := range services {
		servicesIndex[s.Topic()] = s
		thingInclusionReport.Services = append(thingInclusionReport.Services, *s.Specification())
	}

	return &thing{
		inclusionReport: thingInclusionReport,
		services:        servicesIndex,
	}
}

// thing is a private implementation of a FIMP thing.
type thing struct {
	inclusionReport *fimptype.ThingInclusionReport
	services        map[string]Service
}

// InclusionReport returns an inclusion report of the thing.
func (t *thing) InclusionReport() *fimptype.ThingInclusionReport {
	return t.inclusionReport
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

// ServiceByTopic returns a service based on the topic on which is is supposed to be listening for commands.
func (t *thing) ServiceByTopic(topic string) Service {
	for serviceTopic, s := range t.services {
		if strings.HasSuffix(topic, serviceTopic) {
			return s
		}
	}

	return nil
}
