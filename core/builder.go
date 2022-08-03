package core

import (
	"errors"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/discovery"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

// NewBuilder creates new instance of a core app builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Builder is a core app builder that helps to set up and run core application on a hub.
type Builder struct {
	mqtt               *fimpgo.MqttTransport
	resource           *discovery.Resource
	topicSubscriptions []string
	routing            []*router.Routing
	tasks              []*task.Task
	services           []Service
}

// WithMQTT sets the MQTT broker.
func (b *Builder) WithMQTT(mqtt *fimpgo.MqttTransport) *Builder {
	b.mqtt = mqtt

	return b
}

// WithTopicSubscription sets topic that should be subscribed to.
func (b *Builder) WithTopicSubscription(topicSubscriptions ...string) *Builder {
	b.topicSubscriptions = append(b.topicSubscriptions, topicSubscriptions...)

	return b
}

// WithRouting sets MQTT topic routing.
func (b *Builder) WithRouting(routing ...*router.Routing) *Builder {
	b.routing = append(b.routing, routing...)

	return b
}

// WithTask sets background task to be performed.
func (b *Builder) WithTask(tasks ...*task.Task) *Builder {
	b.tasks = append(b.tasks, tasks...)

	return b
}

// WithServiceDiscovery sets the optional service discovery resource.
func (b *Builder) WithServiceDiscovery(resource *discovery.Resource) *Builder {
	b.resource = resource

	return b
}

// WithServices sets the application services.
func (b *Builder) WithServices(services ...Service) *Builder {
	b.services = append(b.services, services...)

	return b
}

// Build builds the core application.
func (b *Builder) Build() (Core, error) {
	if err := b.check(); err != nil {
		return nil, err
	}

	if b.resource != nil {
		b.topicSubscriptions = append(b.topicSubscriptions, discovery.Topic)
		b.routing = append(b.routing, discovery.Route(b.resource))
	}

	messageRouter := router.NewRouter(b.mqtt, router.DefaultChannelID, b.routing...)

	taskManager := task.NewManager(b.tasks...)

	return &core{
		mqtt:               b.mqtt,
		topicSubscriptions: b.topicSubscriptions,
		messageRouter:      messageRouter,
		taskManager:        taskManager,
		services:           b.services,
	}, nil
}

// check performs checks if all required components have been provided to the builder.
func (b *Builder) check() error {
	if b.mqtt == nil {
		return errors.New("core app builder: it is required to provide MQTT broker instance")
	}

	return nil
}
