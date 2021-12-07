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
func NewRouter(mqt *fimpgo.MqttTransport, channelID string, routing ...*Routing) Router {
	return &router{
		channelID: channelID,
		routing:   routing,
		mqtt:      mqt,
		lock:      &sync.Mutex{},
		wg:        &sync.WaitGroup{},
	}
}

// router is an implementation of the router service.
type router struct {
	channelID string
	routing   []*Routing
	mqtt      *fimpgo.MqttTransport
	lock      *sync.Mutex
	wg        *sync.WaitGroup
	stopCh    chan struct{}
}

// Start starts the router and initiates processing of incoming messages.
func (r *router) Start() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.stopCh != nil {
		return errors.New("message router: cannot be started as it is already running")
	}

	r.stopCh = make(chan struct{})
	messageCh := make(fimpgo.MessageCh, 5)
	r.mqtt.RegisterChannel(r.channelID, messageCh)

	go r.routeMessages(messageCh)

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

	r.stopCh = nil

	return nil
}

// routeMessages routes incoming message.
func (r *router) routeMessages(messageCh fimpgo.MessageCh) {
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
	for _, routing := range r.routing {
		if !routing.vote(message) {
			continue
		}

		response := routing.handler.Handle(message)
		if response == nil {
			continue
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
