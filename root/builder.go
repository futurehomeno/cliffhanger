package root

import (
	"errors"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/config"
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

type mqttFunc func(*fimpgo.MqttTransport)
type resourceFunc func(*discovery.Resource)

// Builder is a root app builder that helps to set up and run root application on a hub.
type Builder struct {
	edge               bool
	topicSubscriptions []string
	routing            []*router.Routing
	routerOptions      []router.Option
	tasks              []*task.Task
	services           []Service
	resetters          []Resetter

	mqttFunc     mqttFunc
	resourceFunc resourceFunc
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

func (b *Builder) ConfigureMqtt(f mqttFunc) *Builder {
	b.mqttFunc = f
	return b
}

func (b *Builder) ConfigureResource(f resourceFunc) *Builder {
	b.resourceFunc = f
	return b
}

// Build assembles the root application.
func (b *Builder) Build(resourceName string, cfg *config.Default) App {
	app := &app{
		errCh:       make(chan error),
		mqtt:        defaultMqtt(cfg),
		taskManager: task.NewManager(b.tasks...),
		services:    b.services,
		resetters:   b.resetters,
	}
	if b.edge {
		app.lifecycle = lifecycle.New()
	}
	if b.mqttFunc != nil {
		b.mqttFunc(app.mqtt)
	}

	b.prepareRouting(resourceName, app)

	return app
}

// prepareRouting prepares routing for the root application.
func (b *Builder) prepareRouting(resourceName string, rootApp *app) {
	var resource *discovery.Resource
	if b.edge {
		resource = discovery.EdgeResource(resourceName)
	} else {
		resource = discovery.CoreResource(resourceName)
	}
	if b.resourceFunc != nil {
		b.resourceFunc(resource)
	}
	routing := append(b.routing, discovery.Route(resource))
	topicSubscriptions := append(b.topicSubscriptions, discovery.Topic)

	// Include application factory reset routing only if resetters are provided.
	if len(rootApp.resetters) > 0 {
		topicSubscriptions = append(topicSubscriptions, GatewayEvtTopic)
		routing = append(routing, routeFactoryReset(rootApp))
	}

	rootApp.topicSubscriptions = topicSubscriptions
	rootApp.messageRouter = router.NewRouter(rootApp.mqtt, router.DefaultChannelID, routing...).WithOptions(b.routerOptions...)
}

func defaultMqtt(cfg *config.Default) *fimpgo.MqttTransport {
	return fimpgo.NewMqttTransport(
		cfg.MQTTServerURI,
		cfg.MQTTClientIDPrefix,
		cfg.MQTTUsername,
		cfg.MQTTPassword,
		true,
		1,
		1,
	)
}
