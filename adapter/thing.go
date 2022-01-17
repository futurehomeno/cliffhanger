package adapter

import (
	"strings"

	"github.com/futurehomeno/fimpgo/fimptype"
)

// Thing is an interface representing FIMP thing.
type Thing interface {
	// InclusionReport returns an inclusion report of the thing.
	InclusionReport() *fimptype.ThingInclusionReport
	// Address returns address of the thing.
	Address() string
	// Services returns all services from the thing that match the provided name. If empty all services are returned.
	Services(name string) []Service
	// ServiceTopics returns all service topics.
	ServiceTopics() []string
	// ServiceByTopic returns a service based on the topic on which is is supposed to be listening for commands.
	ServiceByTopic(topic string) Service
}

// NewThing creates new instance of a FIMP thing.
func NewThing(thingInclusionReport *fimptype.ThingInclusionReport, services ...Service) Thing {
	topicIndex := make(map[string]Service)
	thingInclusionReport.Services = nil

	for _, s := range services {
		topicIndex[s.Topic()] = s
		thingInclusionReport.Services = append(thingInclusionReport.Services, *s.Specification())
	}

	return &thing{
		thingInclusionReport: thingInclusionReport,
		topicIndex:           topicIndex,
	}
}

// thing is a private implementation of a FIMP thing.
type thing struct {
	thingInclusionReport *fimptype.ThingInclusionReport
	topicIndex           map[string]Service
}

// InclusionReport returns an inclusion report of the thing.
func (t *thing) InclusionReport() *fimptype.ThingInclusionReport {
	return t.thingInclusionReport
}

// Address returns address of the thing.
func (t *thing) Address() string {
	return t.thingInclusionReport.Address
}

// Services returns all services from the thing that match the provided name. If empty all services are returned.
func (t *thing) Services(name string) []Service {
	var services []Service

	for _, s := range t.topicIndex {
		if name != "" && s.Name() != name {
			continue
		}

		services = append(services, s)
	}

	return services
}

// ServiceTopics returns all service topics.
func (t *thing) ServiceTopics() []string {
	var topics []string

	for topic := range t.topicIndex {
		topics = append(topics, topic)
	}

	return topics
}

// ServiceByTopic returns a service based on the topic on which is is supposed to be listening for commands.
func (t *thing) ServiceByTopic(topic string) Service {
	for serviceTopic, s := range t.topicIndex {
		if strings.HasSuffix(topic, serviceTopic) {
			return s
		}
	}

	return nil
}
