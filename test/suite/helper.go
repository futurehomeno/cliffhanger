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

func FloatMessage(topic, messageType, service string, value float64) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewFloatMessage(
			messageType,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func ObjectMessage(topic, messageType, service string, value interface{}) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewObjectMessage(
			messageType,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func StringMapMessage(topic, messageType, service string, value map[string]string) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewStrMapMessage(
			messageType,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}
