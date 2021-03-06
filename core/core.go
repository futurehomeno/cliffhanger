package core

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

// Core is an interface representing an core application service.
type Core interface {
	// Start starts the core application.
	Start() error
	// Stop stops the core application maintaining a graceful shutdown.
	Stop() error
}

// New creates a new core application instance.
func New(
	mqtt *fimpgo.MqttTransport,
	topicSubscriptions []string,
	router router.Router,
	taskManager task.Manager,
) Core {
	return &core{
		mqtt:               mqtt,
		topicSubscriptions: topicSubscriptions,
		messageRouter:      router,
		taskManager:        taskManager,
	}
}

// core is an implementation of core application interface.
type core struct {
	mqtt               *fimpgo.MqttTransport
	topicSubscriptions []string
	messageRouter      router.Router
	taskManager        task.Manager
}

// Start starts the core application.
func (e *core) Start() error {
	err := e.mqtt.Start()
	if err != nil {
		return fmt.Errorf("core: failed to start MQTT broker: %w", err)
	}

	err = e.messageRouter.Start()
	if err != nil {
		return fmt.Errorf("core: failed to start message router: %w", err)
	}

	for _, topic := range e.topicSubscriptions {
		err = e.mqtt.Subscribe(topic)
		if err != nil {
			return fmt.Errorf("core: failed to subscribe to a topic %s: %w", topic, err)
		}
	}

	err = e.taskManager.Start()
	if err != nil {
		return fmt.Errorf("core: failed to start task manager: %w", err)
	}

	return nil
}

// Stop stops the core application maintaining a graceful shutdown.
func (e *core) Stop() error {
	err := e.taskManager.Stop()
	if err != nil {
		return fmt.Errorf("core: failed to stop task manager: %w", err)
	}

	for _, topic := range e.topicSubscriptions {
		err := e.mqtt.Unsubscribe(topic)
		if err != nil {
			return fmt.Errorf("core: failed to unsubscribe to a topic %s: %w", topic, err)
		}
	}

	err = e.messageRouter.Stop()
	if err != nil {
		return fmt.Errorf("core: failed to stop message router: %w", err)
	}

	e.mqtt.Stop()

	return nil
}
