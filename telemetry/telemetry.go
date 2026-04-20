package telemetry

import (
	"errors"
	"fmt"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
)

const (
	// Topic is the default FIMP topic used by the cloud telemetry pipeline.
	// Picking mt:rsp is deliberate: it matches the existing CloudBridge
	// LocalToCloud default route so no bridge change is needed.
	Topic       = "pt:j1/mt:rsp/rt:cloud/rn:backend-service/ad:telemetry"
	MessageType = "evt.telemetry.report"
	Service     = "telemetry"
)

// Event is the payload carried in the FIMP val field.
type Event struct {
	Event  string         `json:"event"`
	Domain string         `json:"domain,omitempty"`
	Data   map[string]any `json:"data,omitempty"`
}

// Reporter emits telemetry events over MQTT to the cloud backend-service.
type Reporter interface {
	// Report publishes an event with the given name, optional domain, and
	// free-form data payload.
	Report(event, domain string, data map[string]any) error
	// ReportEvent publishes a pre-built Event.
	ReportEvent(event *Event) error
	// SetTargetTopic overrides the default target topic. Passing an empty
	// string restores the default.
	SetTargetTopic(topic string)
}

// New returns a Reporter that publishes telemetry events as the given source.
// The source becomes the FIMP src field and is used by the consumer to populate
// the app column, so it must uniquely identify the emitting application.
func New(mqtt *fimpgo.MqttTransport, source string) (Reporter, error) {
	if mqtt == nil {
		return nil, errors.New("telemetry: mqtt transport is required")
	}

	if source == "" {
		return nil, errors.New("telemetry: source is required")
	}

	return &reporter{
		mqtt:   mqtt,
		source: fimptype.ResourceNameT(source),
		topic:  Topic,
	}, nil
}

type reporter struct {
	mqtt   *fimpgo.MqttTransport
	source fimptype.ResourceNameT

	mu    sync.RWMutex
	topic string
}

func (r *reporter) Report(event, domain string, data map[string]any) error {
	return r.ReportEvent(&Event{
		Event:  event,
		Domain: domain,
		Data:   data,
	})
}

func (r *reporter) ReportEvent(event *Event) error {
	if event == nil || event.Event == "" {
		return errors.New("telemetry: event name is required")
	}

	msg := fimpgo.NewObjectMessage(MessageType, Service, event, nil, nil, nil)
	msg.Source = r.source

	r.mu.RLock()
	topic := r.topic
	r.mu.RUnlock()

	if err := r.mqtt.PublishToTopic(topic, msg); err != nil {
		return fmt.Errorf("telemetry: failed to publish event: %w", err)
	}

	return nil
}

func (r *reporter) SetTargetTopic(topic string) {
	if topic == "" {
		topic = Topic
	}

	r.mu.Lock()
	r.topic = topic
	r.mu.Unlock()
}
