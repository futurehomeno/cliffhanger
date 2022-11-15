package notification

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
)

type TimelineEvent struct {
	EventName      string `json:"EventName"`
	MessageContent string `json:"MessageContent"`
	DeviceID       int    `json:"DeviceId"`
	DeviceName     string `json:"DeviceName"`
	RoomName       string `json:"RoomName"`
	AreaName       string `json:"AreaName"`
}

type TimelineMessage struct {
	Sender    string `json:"sender"`
	MessageEN string `json:"message_en"`
	MessageNO string `json:"message_no"`
}

type Timeline interface {
	Event(event *TimelineEvent) error
	Message(message *TimelineMessage) error
}

func NewTimeline(mqtt *fimpgo.MqttTransport) Timeline {
	return &timeline{
		mqtt: mqtt,
	}
}

type timeline struct {
	mqtt *fimpgo.MqttTransport
}

func (t *timeline) Event(event *TimelineEvent) error {
	payload := map[string]string{
		"EventName":      event.EventName,
		"MessageContent": event.MessageContent,
		"DeviceId":       idToString(event.DeviceID),
		"DeviceName":     event.DeviceName,
		"RoomName":       event.RoomName,
		"AreaName":       event.AreaName,
	}

	msg := fimpgo.NewStrMapMessage("cmd.timeline.set", "kind_owl", payload, nil, nil, nil)

	err := t.mqtt.PublishToTopic("pt:j1/mt:cmd/rt:app/rn:time_owl/ad:1", msg)
	if err != nil {
		return fmt.Errorf("failed to send timeline event: %w", err)
	}

	return nil
}

func (t *timeline) Message(message *TimelineMessage) error {
	payload := map[string]string{
		"sender":     message.Sender,
		"message_en": message.MessageEN,
		"message_no": message.MessageNO,
	}

	msg := fimpgo.NewStrMapMessage("cmd.timeline.set", "time_owl", payload, nil, nil, nil)

	err := t.mqtt.PublishToTopic("pt:j1/mt:cmd/rt:app/rn:time_owl/ad:1", msg)
	if err != nil {
		return fmt.Errorf("failed to send timeline event: %w", err)
	}

	return nil
}
