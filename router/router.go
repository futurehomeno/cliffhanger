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
		case <-r.stopCh:
			return
		case message := <-messageCh:
			for _, routing := range r.routing {
				r.processMessage(routing, message)
			}
		}
	}
}

// processMessage executes handlers responsible for processing the incoming message and send response if applicable.
func (r *router) processMessage(routing *Routing, message *fimpgo.Message) {
	defer func() {
		if rc := recover(); rc != nil {
			r.handleProcessingPanic(message, rc)
		}
	}()

	if !routing.vote(message) {
		return
	}

	if r.cfg.processingCallback != nil {
		r.cfg.processingCallback(message)
	}

	response := routing.handler.Handle(message)
	if response == nil {
		return
	}

	responseAddress := r.getResponseAddress(message, response)
	if responseAddress == nil {
		return
	}

	if response.Payload.CorrelationID == "" {
		response.Payload.CorrelationID = message.Payload.UID
	}

	if r.cfg.responseCallback != nil {
		r.cfg.responseCallback(message, response)
	}

	err := r.mqtt.Publish(responseAddress, response.Payload)
	if err != nil {
		log.WithError(err).
			WithField("topic", response.Addr.Serialize()).
			WithField("message", response.Payload).
			Error("failed to publish response")
	}
}

func (r *router) handleProcessingPanic(message *fimpgo.Message, panicErr any) {
	log.WithField("topic", message.Addr.Serialize()).
		WithField("service", message.Payload.Service).
		WithField("type", message.Payload.Type).
		Errorf("message router: panic occurred while processing message: %+v", panicErr)

	if r.cfg.panicCallback != nil {
		r.cfg.panicCallback(message, panicErr)
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
	panicCallback        func(message *fimpgo.Message, panicErr any)
	processingCallback   func(message *fimpgo.Message)
	responseCallback     func(in, out *fimpgo.Message)
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

// WithAsyncProcessing returns an option that sets the number of concurrent workers processing incoming messages.
func WithAsyncProcessing(concurrency int) Option {
	return optionFn(func(cfg *config) {
		if concurrency < 2 {
			return
		}

		cfg.concurrency = concurrency
	})
}

// WithMessageBuffer returns an option that sets the buffer size for incoming messages.
func WithMessageBuffer(buffer int) Option {
	return optionFn(func(cfg *config) {
		if buffer < 0 {
			return
		}

		cfg.buffer = buffer
	})
}

// WithPreservedGlobalPrefix returns an option that enables preserving global prefix in the reply topic address.
func WithPreservedGlobalPrefix() Option {
	return optionFn(func(cfg *config) {
		cfg.preserveGlobalPrefix = true
	})
}

// WithPanicCallback returns an option that sets a callback function that will be called when a panic occurs.
func WithPanicCallback(f func(message *fimpgo.Message, err any)) Option {
	return optionFn(func(cfg *config) {
		cfg.panicCallback = f
	})
}

// WithMessageProcessingCallback returns an option that sets a callback function that will be called when a message is processed.
func WithMessageProcessingCallback(f func(message *fimpgo.Message)) Option {
	return optionFn(func(cfg *config) {
		cfg.processingCallback = f
	})
}

// WithResponseCallback returns an option that sets a callback function that will be called before a response is sent.
func WithResponseCallback(f func(in, out *fimpgo.Message)) Option {
	return optionFn(func(cfg *config) {
		cfg.responseCallback = f
	})
}
