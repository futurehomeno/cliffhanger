package router

import (
	"errors"
	"sync"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
)

// DefaultChannelID is a constant defining a default channel ID used by the router.
const DefaultChannelID = "main_router"

// Router is an interface representing a service responsible for routing messages.
type Router interface {
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
	}
}

// router is an implementation of the router service.
type router struct {
	cfg       *config
	channelID string
	routing   []*Routing
	mqtt      *fimpgo.MqttTransport
	lock      *sync.Mutex
	wg        *sync.WaitGroup
	stopCh    chan struct{}
}

// WithOptions applies options to the router configuration.
func (r *router) WithOptions(options ...Option) Router {
	for _, option := range options {
		option.apply(r.cfg)
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

		response := routing.handler.Handle(message)
		if response == nil {
			continue
		}

		if response.Payload.CorrelationID == "" {
			response.Payload.CorrelationID = message.Payload.UID
		}

		if message.Payload.ResponseToTopic != "" {
			err := r.mqtt.RespondToRequest(message.Payload, response.Payload)
			if err != nil {
				log.WithError(err).
					WithField("topic", message.Payload.ResponseToTopic).
					WithField("message", response.Payload).
					Error("failed to publish response")
			}
		} else if response.Addr != nil {
			err := r.mqtt.Publish(response.Addr, response.Payload)
			if err != nil {
				log.WithError(err).
					WithField("topic", response.Addr.Serialize()).
					WithField("message", response.Payload).
					Error("failed to publish response")
			}
		}
	}
}

func defaultConfig() *config {
	return &config{
		async:       true,
		buffer:      10,
		concurrency: 5,
	}
}

type config struct {
	async       bool
	buffer      int
	concurrency int
}

// Option is an interface representing a message router configuration option.
type Option interface {
	// apply applies option to the message router.
	apply(cfg *config)
}

// messageHandlerOptionFn is an adapter allowing usage of anonymous function as a service meeting message router option interface.
type optionFn func(cfg *config)

// apply applies option to the message router.
func (f optionFn) apply(cfg *config) {
	f(cfg)
}

// WithSyncProcessing returns an option that enables synchronous processing of incoming messages.
func WithSyncProcessing() Option {
	return optionFn(func(cfg *config) {
		cfg.concurrency = 1
	})
}

// WithConcurrency returns an option that sets the number of concurrent workers processing incoming messages.
func WithConcurrency(concurrency int) Option {
	return optionFn(func(cfg *config) {
		if concurrency < 1 {
			concurrency = 1
		}

		cfg.concurrency = concurrency
	})
}

// WithMessageBuffer returns an option that sets the buffer size for incoming messages.
func WithMessageBuffer(buffer int) Option {
	return optionFn(func(cfg *config) {
		if buffer < 0 {
			buffer = 0
		}

		cfg.buffer = buffer
	})
}
