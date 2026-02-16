package suite

import (
	"time"

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

func BoolMessage(topic, messageType, service string, value bool) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewBoolMessage(
			messageType,
			service,
			value,
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

func IntMessage(topic, messageType, service string, value int) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewIntMessage(
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

func FloatMapMessage(topic, messageType, service string, value map[string]float64) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewFloatMapMessage(
			messageType,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func IntMapMessage(topic, messageType, service string, value map[string]int) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewIntMapMessage(
			messageType,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func BoolMapMessage(topic, messageType, service string, value map[string]bool) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewBoolMapMessage(
			messageType,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func StringArrayMessage(topic, messageType, service string, value []string) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewStrArrayMessage(
			messageType,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func FloatArrayMessage(topic, messageType, service string, value []float64) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewFloatArrayMessage(
			messageType,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func IntArrayMessage(topic, messageType, service string, value []int) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewIntArrayMessage(
			messageType,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func BoolArrayMessage(topic, messageType, service string, value []bool) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewBoolArrayMessage(
			messageType,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{
		props: make(fimpgo.Props),
		tags:  make(fimpgo.Tags, 0),
	}
}

type MessageBuilder struct {
	msg   *fimpgo.Message
	props fimpgo.Props
	tags  fimpgo.Tags
}

func (b *MessageBuilder) NullMessage(topic, messageType, service string) *MessageBuilder {
	b.msg = NullMessage(topic, messageType, service)

	return b
}

func (b *MessageBuilder) BoolMessage(topic, messageType, service string, value bool) *MessageBuilder {
	b.msg = BoolMessage(topic, messageType, service, value)

	return b
}

func (b *MessageBuilder) StringMessage(topic, messageType, service, value string) *MessageBuilder {
	b.msg = StringMessage(topic, messageType, service, value)

	return b
}

func (b *MessageBuilder) IntMessage(topic, messageType, service string, value int) *MessageBuilder {
	b.msg = IntMessage(topic, messageType, service, value)

	return b
}

func (b *MessageBuilder) FloatMessage(topic, messageType, service string, value float64) *MessageBuilder {
	b.msg = FloatMessage(topic, messageType, service, value)

	return b
}

func (b *MessageBuilder) ObjectMessage(topic, messageType, service string, value interface{}) *MessageBuilder {
	b.msg = ObjectMessage(topic, messageType, service, value)

	return b
}

func (b *MessageBuilder) StringMapMessage(topic, messageType, service string, value map[string]string) *MessageBuilder {
	b.msg = StringMapMessage(topic, messageType, service, value)

	return b
}

func (b *MessageBuilder) FloatMapMessage(topic, messageType, service string, value map[string]float64) *MessageBuilder {
	b.msg = FloatMapMessage(topic, messageType, service, value)

	return b
}

func (b *MessageBuilder) IntMapMessage(topic, messageType, service string, value map[string]int) *MessageBuilder {
	b.msg = IntMapMessage(topic, messageType, service, value)

	return b
}

func (b *MessageBuilder) BoolMapMessage(topic, messageType, service string, value map[string]bool) *MessageBuilder {
	b.msg = BoolMapMessage(topic, messageType, service, value)

	return b
}

func (b *MessageBuilder) StringArrayMessage(topic, messageType, service string, value []string) *MessageBuilder {
	b.msg = StringArrayMessage(topic, messageType, service, value)

	return b
}

func (b *MessageBuilder) FloatArrayMessage(topic, messageType, service string, value []float64) *MessageBuilder {
	b.msg = FloatArrayMessage(topic, messageType, service, value)

	return b
}

func (b *MessageBuilder) IntArrayMessage(topic, messageType, service string, value []int) *MessageBuilder {
	b.msg = IntArrayMessage(topic, messageType, service, value)

	return b
}

func (b *MessageBuilder) BoolArrayMessage(topic, messageType, service string, value []bool) *MessageBuilder {
	b.msg = BoolArrayMessage(topic, messageType, service, value)

	return b
}

func (b *MessageBuilder) AddProperty(key, value string) *MessageBuilder {
	b.props[key] = value

	return b
}

func (b *MessageBuilder) AddTag(t string) *MessageBuilder {
	b.tags = append(b.tags, t)

	return b
}

func (b *MessageBuilder) SetCreationTime(t time.Time) *MessageBuilder {
	b.msg.Payload.CreationTime = t.Format(fimpgo.TimeFormat)

	return b
}

func (b *MessageBuilder) Build() *fimpgo.Message {
	if len(b.props) > 0 {
		if b.msg.Payload.Properties == nil {
			b.msg.Payload.Properties = make(fimpgo.Props)
		}

		for k, v := range b.props {
			b.msg.Payload.Properties[k] = v
		}
	}

	if len(b.tags) > 0 {
		b.msg.Payload.Tags = append(b.msg.Payload.Tags, b.tags...)
	}

	return b.msg
}
