package notification

import (
	"fmt"
	"strconv"

	"github.com/futurehomeno/fimpgo"
)

const CustomEventName = "custom"

type Event struct {
	EventName      string
	MessageContent string
	DeviceID       int
	DeviceName     string
	RoomID         int
	RoomName       string
	AreaID         int
	AreaName       string
	AreaType       string
}

type Notification interface {
	Event(event *Event) error
	Message(message string) error
}

func NewNotification(mqtt *fimpgo.MqttTransport) Notification {
	return &notification{
		mqtt: mqtt,
	}
}

type notification struct {
	mqtt *fimpgo.MqttTransport
}

// Event sends a push notification event.
func (n *notification) Event(event *Event) error {
	payload := map[string]string{
		"EventName":      event.EventName,
		"MessageContent": event.MessageContent,
		"DeviceId":       idToString(event.DeviceID),
		"DeviceName":     event.DeviceName,
		"RoomId":         idToString(event.RoomID),
		"RoomName":       event.RoomName,
		"AreaId":         idToString(event.AreaID),
		"AreaName":       event.AreaName,
		"AreaType":       event.AreaType,
	}

	message := fimpgo.NewStrMapMessage("evt.notification.report", "kind_owl", payload, nil, nil, nil)

	err := n.mqtt.PublishToTopic("pt:j1/mt:evt/rt:app/rn:kind_owl/ad:1", message)
	if err != nil {
		return fmt.Errorf("notification: failed to send a notification event: %w", err)
	}

	return nil
}

// Message sends custom push notification event with the provided message.
func (n *notification) Message(message string) error {
	return n.Event(&Event{
		EventName:      CustomEventName,
		MessageContent: message,
	})
}

func idToString(id int) string {
	if id == 0 {
		return ""
	}

	return strconv.Itoa(id)
}
