package router

import (
	"context"
	"errors"
	"sync"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/tracing"
)

const (
	// DefaultChannelID is a constant defining a default channel ID used by the router.
	DefaultChannelID = "main_router"
)

// Router is an interface representing a service responsible for routing messages.
type Router interface {
	// WithOptions applies options to the router configuration.
	WithOptions(options ...Option) Router
	// Start starts the router and initiates processing of incoming messages.
	Start() error
	// Stop stops the router and interrupts processing of incoming messages.
	Stop() error
}

// NewRouter creates new instance of a router service.
func NewRouter(mqtt *fimpgo.MqttTransport, channelID string, routing ...*Routing) Router {
	return &router{
		channelID: channelID,
		routing:   routing,
		mqtt:      mqtt,
		lock:      &sync.Mutex{},
		wg:        &sync.WaitGroup{},
		cfg:       defaultConfig(),
		tracer:    tracing.NewNoOpTracer(),
	}
}

// router is an implementation of the router service.
type router struct {
	cfg       *config
	channelID string
	routing   []*Routing
	mqtt      *fimpgo.MqttTransport
	tracer    tracing.Tracer
	lock      *sync.Mutex
	wg        *sync.WaitGroup
	stopCh    chan struct{}
}

// WithOptions applies options to the router configuration.
func (r *router) WithOptions(options ...Option) Router {
	for _, o := range options {
		o(r)
	}

	return r
}

// Start starts the router and initiates processing of incoming messages.
func (r *router) Start() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.stopCh != nil {
		return errors.New("message router: cannot be started as it is already running")
	}

	r.stopCh = make(chan struct{})
	messageCh := make(fimpgo.MessageCh, r.cfg.buffer)
	r.mqtt.RegisterChannel(r.channelID, messageCh)

	r.wg.Add(r.cfg.concurrency)

	for i := 0; i < r.cfg.concurrency; i++ {
		go r.routeMessages(messageCh)
	}

	return nil
}

// Stop stops the router and interrupts processing of incoming messages.
func (r *router) Stop() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.stopCh == nil {
		return errors.New("message router: cannot be stopped as it is already not running")
	}

	r.mqtt.UnregisterChannel(r.channelID)
	close(r.stopCh)

	r.wg.Wait()

	r.stopCh = nil

	return nil
}

// routeMessages routes incoming message.
func (r *router) routeMessages(messageCh fimpgo.MessageCh) {
	defer r.wg.Done()

	for {
		select {
		case message := <-messageCh:
			r.processMessage(message)
		case <-r.stopCh:
			return
		}
	}
}

// processMessage executes handlers responsible for processing the incoming message and send response if applicable.
func (r *router) processMessage(message *fimpgo.Message) {
	defer func() {
		if r := recover(); r != nil {
			log.WithField("topic", message.Addr.Serialize()).
				WithField("service", message.Payload.Service).
				WithField("type", message.Payload.Type).
				Errorf("message router: panic occurred while processing message: %+v", r)
		}
	}()

	for _, routing := range r.routing {
		if !routing.vote(message) {
			continue
		}

		span, ctx := r.tracer.StartSpanFromContext(context.Background(), OperationNameRouteMessage, tracing.WithSpanType(SpanTypeMqtt))
		span.SetTag(TagMessageType, message.Payload.Type)
		span.SetTag(TagMessageService, message.Payload.Service)
		span.SetTag(TagMessageTopic, message.Topic)

		if message.Payload.Source != "" {
			span.SetTag(TagMessageSource, message.Payload.Source)
		}

		response := routing.handler.Handle(ctx, message)
		if response == nil {
			span.Finish()

			continue
		}

		if response.Payload.Type == EvtErrorReport {
			span.SetTag(TagError, response.Payload.Properties[PropertyMsg])
		}

		span.Finish()

		responseAddress := r.getResponseAddress(message, response)
		if responseAddress == nil {
			continue
		}

		if response.Payload.CorrelationID == "" {
			response.Payload.CorrelationID = message.Payload.UID
		}

		err := r.mqtt.Publish(responseAddress, response.Payload)
		if err != nil {
			log.WithError(err).
				WithField("topic", response.Addr.Serialize()).
				WithField("message", response.Payload).
				Error("failed to publish response")
		}
	}
}

// getResponseAddress returns an address to which the response should be sent based on the incoming message and response.
func (r *router) getResponseAddress(message, response *fimpgo.Message) *fimpgo.Address {
	if message.Payload.ResponseToTopic == "" && response.Addr == nil {
		return nil
	}

	var (
		err             error
		responseAddress *fimpgo.Address
	)

	if message.Payload.ResponseToTopic != "" {
		responseAddress, err = fimpgo.NewAddressFromString(message.Payload.ResponseToTopic)
		if err != nil {
			log.WithError(err).
				WithField("topic", message.Addr.Serialize()).
				WithField("message", message).
				Error("failed to parse respond to topic address")

			return nil
		}
	} else {
		responseAddress = response.Addr
	}

	if r.cfg.preserveGlobalPrefix {
		responseAddress.GlobalPrefix = message.Addr.GlobalPrefix
	}

	return responseAddress
}

// defaultConfig returns a default configuration of the message router.
func defaultConfig() *config {
	return &config{
		buffer:               10,
		concurrency:          5,
		preserveGlobalPrefix: false,
	}
}

// config is a configuration of the message router.
type config struct {
	buffer               int
	concurrency          int
	preserveGlobalPrefix bool
}

// Option represents an option for the router.
type Option func(r *router)

// WithSyncProcessing returns an option that enables synchronous processing of incoming messages.
func WithSyncProcessing() Option {
	return func(r *router) {
		r.cfg.concurrency = 1
	}
}

// WithAsyncProcessing returns an option that sets the number of concurrent workers processing incoming messages.
func WithAsyncProcessing(concurrency int) Option {
	return func(r *router) {
		if concurrency < 2 {
			return
		}

		r.cfg.concurrency = concurrency
	}
}

// WithMessageBuffer returns an option that sets the buffer size for incoming messages.
func WithMessageBuffer(buffer int) Option {
	return func(r *router) {
		if buffer < 0 {
			return
		}

		r.cfg.buffer = buffer
	}
}

// WithPreservedGlobalPrefix returns an option that enables preserving global prefix in the reply topic address.
func WithPreservedGlobalPrefix() Option {
	return func(r *router) {
		r.cfg.preserveGlobalPrefix = true
	}
}

// WithTracer returns an option that sets the tracer for the router.
func WithTracer(t tracing.Tracer) Option {
	return func(r *router) {
		r.tracer = t
	}
}
