package suite

import (
	"github.com/futurehomeno/fimpgo"
)

func NullMessage(topic, messageType, service string) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewNullMessage(
			messageType,
			service,
			nil,
			nil,
			nil,
		),
	}
}

func StringMessage(topic, messageType, service, value string) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewStringMessage(
			messageType,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}