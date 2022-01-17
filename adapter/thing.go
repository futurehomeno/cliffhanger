package adapter

import (
	"strings"

	"github.com/futurehomeno/fimpgo/fimptype"
)

type Thing interface {
	InclusionReport() *fimptype.ThingInclusionReport
	Address() string
	Services(name string) []Service
	ServiceTopics() []string
	ServiceByTopic(topic string) Service
}

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

type thing struct {
	thingInclusionReport *fimptype.ThingInclusionReport
	topicIndex           map[string]Service
}

func (t *thing) InclusionReport() *fimptype.ThingInclusionReport {
	return t.thingInclusionReport
}

func (t *thing) Address() string {
	return t.thingInclusionReport.Address
}

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

func (t *thing) ServiceTopics() []string {
	var topics []string

	for topic := range t.topicIndex {
		topics = append(topics, topic)
	}

	return topics
}

func (t *thing) ServiceByTopic(topic string) Service {
	for serviceTopic, s := range t.topicIndex {
		if strings.HasSuffix(topic, serviceTopic) {
			return s
		}
	}

	return nil
}
