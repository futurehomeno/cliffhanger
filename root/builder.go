package root

import (
	"errors"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/discovery"
	"github.com/futurehomeno/cliffhanger/lifecycle"

	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/cliffhanger/telemetry"
)

func NewEdgeAppBuilder() *Builder {
	return newBuilder(true)
}

func NewCoreAppBuilder() *Builder {
	return newBuilder(false)
}

func newBuilder(edge bool) *Builder {
	return &Builder{
		edge: edge,
	}
}

// Builder is a root app builder that helps to set up and run root application on a hub.
type Builder struct {
	edge               bool
	mqtt               *fimpgo.MqttTransport
	resourceName       fimptype.ResourceNameT
	resourceType       fimptype.ResourceTypeT
	packageName        string
	instanceID         string
	version            string
	lifecycle          *lifecycle.Lifecycle
	telemetry          telemetry.Telemetry
	topicSubscriptions []string
	routing            []*router.Routing
	routerOptions      []router.Option
	tasks              []*task.Task
	services           []Service
	resetters          []Resetter
}

func (b *Builder) WithMQTT(mqtt *fimpgo.MqttTransport) *Builder {
	b.mqtt = mqtt
	return b
}

func (b *Builder) WithServiceDiscovery(resourceName fimptype.ResourceNameT, resourceType fimptype.ResourceTypeT,
	packageName, instanceID, version string) *Builder {
	b.resourceName = resourceName
	b.resourceType = resourceType
	b.packageName = packageName
	b.instanceID = instanceID
	b.version = version
	return b
}

func (b *Builder) WithLifecycle(l *lifecycle.Lifecycle) *Builder {
	b.lifecycle = l
	return b
}

func (b *Builder) WithTelemetry(t telemetry.Telemetry) *Builder {
	b.telemetry = t
	return b
}

func (b *Builder) WithTopicSubscription(topicSubscriptions ...string) *Builder {
	b.topicSubscriptions = append(b.topicSubscriptions, topicSubscriptions...)
	return b
}

func (b *Builder) WithRouterOptions(options ...router.Option) *Builder {
	b.routerOptions = append(b.routerOptions, options...)
	return b
}

func (b *Builder) WithRouting(routing ...*router.Routing) *Builder {
	b.routing = append(b.routing, routing...)
	return b
}

func (b *Builder) WithTask(tasks ...*task.Task) *Builder {
	b.tasks = append(b.tasks, tasks...)
	return b
}

func (b *Builder) WithServices(services ...Service) *Builder {
	b.services = append(b.services, services...)
	return b
}

func (b *Builder) WithResetter(resetter ...Resetter) *Builder {
	b.resetters = append(b.resetters, resetter...)
	return b
}

func (b *Builder) Build() (App, error) {
	if err := b.check(); err != nil {
		return nil, err
	}

	return b.doBuild(), nil
}

func (b *Builder) doBuild() App {
	rootApp := &app{
		lock:  &sync.Mutex{},
		errCh: make(chan error),

		mqtt:         b.mqtt,
		lifecycle:    b.lifecycle,
		telemetry:    b.telemetry,
		resourceName: b.resourceName,
		taskManager:  task.NewManager(b.tasks...),
		services:     b.services,
		resetters:    b.resetters,
	}

	b.prepareRouting(rootApp)

	return rootApp
}

// prepareRouting prepares routing for the root application.
func (b *Builder) prepareRouting(rootApp *app) {
	topicSubscriptions := append(b.topicSubscriptions, discovery.Topic) //nolint:gocritic
	routing := append(                                                  //nolint:gocritic
		b.routing,
		discovery.Route(
			b.resourceName,
			b.resourceType,
			b.packageName,
			b.instanceID,
			b.version,
			b.lifecycle,
		),
	)

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

	if b.resourceName == "" {
		return errors.New("builder: it is required to provide service discovery resource name")
	}

	if b.resourceType == "" {
		return errors.New("builder: it is required to provide service discovery resource type")
	}

	if b.packageName == "" {
		return errors.New("builder: it is required to provide service discovery resource package")
	}

	if b.instanceID == "" {
		return errors.New("builder: it is required to provide service discovery resource instance")
	}

	if b.version == "" {
		return errors.New("builder: it is required to provide service discovery resource version")
	}

	if b.edge && b.lifecycle == nil {
		return errors.New("builder: it is required for an edge app to provide a lifecycle service instance")
	}

	if !b.edge && b.lifecycle != nil {
		return errors.New("builder: it is not allowed for a core app to provide a lifecycle service instance")
	}

	return nil
}
