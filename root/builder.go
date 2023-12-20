package root

import (
	"errors"
	"sync"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/discovery"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

// NewEdgeAppBuilder creates new instance of an edge app builder.
func NewEdgeAppBuilder() *Builder {
	return newBuilder(true)
}

// NewCoreAppBuilder creates new instance of an core app builder.
func NewCoreAppBuilder() *Builder {
	return newBuilder(false)
}

// newBuilder creates new root app builder instance.
func newBuilder(edge bool) *Builder {
	return &Builder{
		edge: edge,
	}
}

// Builder is a root app builder that helps to set up and run root application on a hub.
type Builder struct {
	edge               bool
	mqtt               *fimpgo.MqttTransport
	resource           *discovery.Resource
	lifecycle          *lifecycle.Lifecycle
	topicSubscriptions []string
	routing            []*router.Routing
	routerOptions      []router.Option
	tasks              []*task.Task
	services           []Service
	resetters          []Resetter
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

// WithLifecycle sets the lifecycle service. Required only for building edge application.
func (b *Builder) WithLifecycle(l *lifecycle.Lifecycle) *Builder {
	b.lifecycle = l

	return b
}

// WithTopicSubscription sets topic that should be subscribed to.
func (b *Builder) WithTopicSubscription(topicSubscriptions ...string) *Builder {
	b.topicSubscriptions = append(b.topicSubscriptions, topicSubscriptions...)

	return b
}

// WithRouterOptions sets router options.
func (b *Builder) WithRouterOptions(options ...router.Option) *Builder {
	b.routerOptions = append(b.routerOptions, options...)

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

// WithServices sets the application services.
func (b *Builder) WithServices(services ...Service) *Builder {
	b.services = append(b.services, services...)

	return b
}

func (b *Builder) WithResetter(resetter ...Resetter) *Builder {
	b.resetters = append(b.resetters, resetter...)

	return b
}

// Build builds the root application.
func (b *Builder) Build() (App, error) {
	if err := b.check(); err != nil {
		return nil, err
	}

	return b.doBuild(), nil
}

// doBuild assembles the root application.
func (b *Builder) doBuild() App {

	rootApp := &app{
		lock:  &sync.Mutex{},
		errCh: make(chan error),

		mqtt:        b.mqtt,
		lifecycle:   b.lifecycle,
		taskManager: task.NewManager(b.tasks...),
		services:    b.services,
		resetters:   b.resetters,
	}

	b.prepareRouting(rootApp)

	return rootApp
}

// prepareRouting prepares routing for the root application.
func (b *Builder) prepareRouting(rootApp *app) {
	topicSubscriptions := append(b.topicSubscriptions, discovery.Topic) //nolint:gocritic
	routing := append(b.routing, discovery.Route(b.resource))           //nolint:gocritic

	// Include application factory reset routing only if resetters are provided.
	if len(rootApp.resetters) > 0 {
		topicSubscriptions = append(topicSubscriptions, GatewayEvtTopic)
		routing = append(routing, routeFactoryReset(rootApp))
	}

	rootApp.topicSubscriptions = topicSubscriptions
	rootApp.messageRouter = router.NewRouter(b.mqtt, router.DefaultChannelID, routing...).WithOptions(b.routerOptions...)
}

// check performs checks if all required components have been provided to the builder.
func (b *Builder) check() error {
	if b.mqtt == nil {
		return errors.New("builder: it is required to provide MQTT broker instance")
	}

	if b.resource == nil {
		return errors.New("builder: it is required to provide service discovery resource instance")
	}

	if b.edge && b.lifecycle == nil {
		return errors.New("builder: it is required for an edge app to provide a lifecycle service instance")
	}

	if !b.edge && b.lifecycle != nil {
		return errors.New("builder: it is not allowed for a core app to provide a lifecycle service instance")
	}

	return nil
}
