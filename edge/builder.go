package edge

import (
	"errors"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/discovery"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

// NewBuilder creates new instance of an edge app builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Builder is an edge app builder that helps to set up and run edge application on a hub.
type Builder struct {
	mqtt               *fimpgo.MqttTransport
	resource           *discovery.Resource
	lifecycle          *lifecycle.Lifecycle
	topicSubscriptions []string
	routing            []*router.Routing
	tasks              []*task.Task
}

// WithMQTT sets the MQTT broker.
func (b *Builder) WithMQTT(mqtt *fimpgo.MqttTransport) *Builder {
	b.mqtt = mqtt

	return b
}

// WithServiceDiscovery sets the service discovery resource.
func (b *Builder) WithServiceDiscovery(resource *discovery.Resource) *Builder {
	b.resource = resource

	return b
}

// WithLifecycle sets the lifecycle service.
func (b *Builder) WithLifecycle(l *lifecycle.Lifecycle) *Builder {
	b.lifecycle = l

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

// Build builds the edge application.
func (b *Builder) Build() (Edge, error) {
	if err := b.check(); err != nil {
		return nil, err
	}

	b.topicSubscriptions = append(b.topicSubscriptions, discovery.Topic)
	b.routing = append(b.routing, discovery.Route(b.resource))

	messageRouter := router.NewRouter(b.mqtt, router.DefaultChannelID, b.routing...)

	taskManager := task.NewManager(b.tasks...)

	return New(
		b.mqtt,
		b.lifecycle,
		b.topicSubscriptions,
		messageRouter,
		taskManager,
	), nil
}

// check checks if all required components have been provided to the builder.
func (b *Builder) check() error {
	if b.mqtt == nil {
		return errors.New("edge app builder: it is required to provide MQTT broker instance")
	}

	if b.resource == nil {
		return errors.New("edge app builder: it is required to provide service discovery resource instance")
	}

	if b.lifecycle == nil {
		return errors.New("edge app builder: it is required to provide lifecycle service instance")
	}

	return nil
}
