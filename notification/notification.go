package notification

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
)

const CustomNotificationType = "custom"

type Notification struct {
	EventName      string `json:"EventName"`
	MessageContent string `json:"MessageContent"`
	SiteID         string `json:"SiteId"`
}

type Manager interface {
	Notification(notificationType string, notificationContent string) error
	Timeline(sender string, languageAndMessage ...string) error
}

func NewNotificationManager(mqtt *fimpgo.MqttTransport, siteID, serviceName string) Manager {
	return &manager{
		mqtt:        mqtt,
		siteID:      siteID,
		serviceName: serviceName,
	}
}

type manager struct {
	mqtt        *fimpgo.MqttTransport
	siteID      string
	serviceName string
}

// Notification sends a notification.
func (m *manager) Notification(notificationType string, notificationContent string) error {
	n := Notification{
		EventName:      notificationType,
		MessageContent: notificationContent,
		SiteID:         m.siteID,
	}

	message := fimpgo.NewObjectMessage("evt.notification.report", "kind-owl", n, nil, nil, nil)
	message.Source = m.serviceName

	err := m.mqtt.PublishToTopic("pt:j1/mt:evt/rt:app/rn:kind_owl/ad:1", message)
	if err != nil {
		return fmt.Errorf("failed to send notification event: %w", err)
	}

	return nil
}

// Timeline sends a timeline event. Example usage:
// 	m.Timeline("My Service", "en", "Timeline in English", "no", "Tidslinje på norsk")
func (m *manager) Timeline(sender string, languageAndMessage ...string) error {
	if len(languageAndMessage)%2 > 0 {
		return fmt.Errorf("odd number of languages and messages")
	}

	e := map[string]string{
		"sender": sender,
	}

	for i := 0; i < len(languageAndMessage); i += 2 {
		e[fmt.Sprintf("message_%s", languageAndMessage[i])] = languageAndMessage[i+1]
	}

	message := fimpgo.NewStrMapMessage("cmd.timeline.set", "kind-owl", e, nil, nil, nil)
	message.Source = m.serviceName

	err := m.mqtt.PublishToTopic("pt:j1/mt:cmd/rt:app/rn:time_owl/ad:1", message)
	if err != nil {
		return fmt.Errorf("failed to send timeline event: %w", err)
	}

	return nil
}
