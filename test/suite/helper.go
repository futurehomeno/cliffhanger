package suite

import (
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
)

func NullMessage(topic, iface string, service fimptype.ServiceNameT) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewNullMessage(
			iface,
			service,
			nil,
			nil,
			nil,
		),
	}
}

func BoolMessage(topic string, iface string, service fimptype.ServiceNameT, value bool) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewBoolMessage(
			iface,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func StringMessage(topic, iface string, service fimptype.ServiceNameT, value string) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewStringMessage(
			iface,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func IntMessage(topic, iface string, service fimptype.ServiceNameT, value int) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewIntMessage(
			iface,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func FloatMessage(topic, iface string, service fimptype.ServiceNameT, value float64) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewFloatMessage(
			iface,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func ObjectMessage(topic, iface string, service fimptype.ServiceNameT, value interface{}) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewObjectMessage(
			iface,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func StringMapMessage(topic, iface string, service fimptype.ServiceNameT, value map[string]string) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewStrMapMessage(
			iface,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func FloatMapMessage(topic, iface string, service fimptype.ServiceNameT, value map[string]float64) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewFloatMapMessage(
			iface,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func IntMapMessage(topic, iface string, service fimptype.ServiceNameT, value map[string]int) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewIntMapMessage(
			iface,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func BoolMapMessage(topic, iface string, service fimptype.ServiceNameT, value map[string]bool) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewBoolMapMessage(
			iface,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func StringArrayMessage(topic, iface string, service fimptype.ServiceNameT, value []string) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewStrArrayMessage(
			iface,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func FloatArrayMessage(topic, iface string, service fimptype.ServiceNameT, value []float64) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewFloatArrayMessage(
			iface,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func IntArrayMessage(topic, iface string, service fimptype.ServiceNameT, value []int) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewIntArrayMessage(
			iface,
			service,
			value,
			nil,
			nil,
			nil,
		),
	}
}

func BoolArrayMessage(topic, iface string, service fimptype.ServiceNameT, value []bool) *fimpgo.Message {
	return &fimpgo.Message{
		Topic: topic,
		Payload: fimpgo.NewBoolArrayMessage(
			iface,
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

func (b *MessageBuilder) NullMessage(topic, iface string, service fimptype.ServiceNameT) *MessageBuilder {
	b.msg = NullMessage(topic, iface, service)

	return b
}

func (b *MessageBuilder) BoolMessage(topic, iface string, service fimptype.ServiceNameT, value bool) *MessageBuilder {
	b.msg = BoolMessage(topic, iface, service, value)

	return b
}

func (b *MessageBuilder) StringMessage(topic, iface string, service fimptype.ServiceNameT, value string) *MessageBuilder {
	b.msg = StringMessage(topic, iface, service, value)

	return b
}

func (b *MessageBuilder) IntMessage(topic, iface string, service fimptype.ServiceNameT, value int) *MessageBuilder {
	b.msg = IntMessage(topic, iface, service, value)

	return b
}

func (b *MessageBuilder) FloatMessage(topic, iface string, service fimptype.ServiceNameT, value float64) *MessageBuilder {
	b.msg = FloatMessage(topic, iface, service, value)

	return b
}

func (b *MessageBuilder) ObjectMessage(topic, iface string, service fimptype.ServiceNameT, value interface{}) *MessageBuilder {
	b.msg = ObjectMessage(topic, iface, service, value)

	return b
}

func (b *MessageBuilder) StringMapMessage(topic, iface string, service fimptype.ServiceNameT, value map[string]string) *MessageBuilder {
	b.msg = StringMapMessage(topic, iface, service, value)

	return b
}

func (b *MessageBuilder) FloatMapMessage(topic, iface string, service fimptype.ServiceNameT, value map[string]float64) *MessageBuilder {
	b.msg = FloatMapMessage(topic, iface, service, value)

	return b
}

func (b *MessageBuilder) IntMapMessage(topic, iface string, service fimptype.ServiceNameT, value map[string]int) *MessageBuilder {
	b.msg = IntMapMessage(topic, iface, service, value)

	return b
}

func (b *MessageBuilder) BoolMapMessage(topic, iface string, service fimptype.ServiceNameT, value map[string]bool) *MessageBuilder {
	b.msg = BoolMapMessage(topic, iface, service, value)

	return b
}

func (b *MessageBuilder) StringArrayMessage(topic, iface string, service fimptype.ServiceNameT, value []string) *MessageBuilder {
	b.msg = StringArrayMessage(topic, iface, service, value)

	return b
}

func (b *MessageBuilder) FloatArrayMessage(topic, iface string, service fimptype.ServiceNameT, value []float64) *MessageBuilder {
	b.msg = FloatArrayMessage(topic, iface, service, value)

	return b
}

func (b *MessageBuilder) IntArrayMessage(topic, iface string, service fimptype.ServiceNameT, value []int) *MessageBuilder {
	b.msg = IntArrayMessage(topic, iface, service, value)

	return b
}

func (b *MessageBuilder) BoolArrayMessage(topic, iface string, service fimptype.ServiceNameT, value []bool) *MessageBuilder {
	b.msg = BoolArrayMessage(topic, iface, service, value)

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
