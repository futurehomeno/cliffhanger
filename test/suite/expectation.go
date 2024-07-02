package suite

import (
	"encoding/json"

	"github.com/futurehomeno/fimpgo"
	"github.com/google/go-cmp/cmp"

	"github.com/futurehomeno/cliffhanger/router"
)

const (
	AtLeastOnce Occurrence = iota
	ExactlyOnce
	AtMostOnce
	Never
)

type Occurrence int

func (o Occurrence) String() string {
	switch o {
	case AtLeastOnce:
		return "at least once"
	case ExactlyOnce:
		return "exactly once"
	case AtMostOnce:
		return "at most once"
	case Never:
		return "never"
	default:
		return "unknown"
	}
}

func ExpectMessage(topic, messageType, service string) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType(messageType).
		ExpectService(service)
}

func ExpectString(topic, messageType, service, value string) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType(messageType).
		ExpectService(service).
		ExpectString(value)
}

func ExpectBool(topic, messageType, service string, value bool) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType(messageType).
		ExpectService(service).
		ExpectBool(value)
}

func ExpectInt(topic, messageType, service string, value int64) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType(messageType).
		ExpectService(service).
		ExpectInt(value)
}

func ExpectFloat(topic, messageType, service string, value float64) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType(messageType).
		ExpectService(service).
		ExpectFloat(value)
}

func ExpectObject(topic, messageType, service string, object interface{}) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType(messageType).
		ExpectService(service).
		ExpectObject(object)
}

func ExpectNull(topic, messageType, service string) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType(messageType).
		ExpectService(service).
		ExpectNull()
}

func ExpectStringMap(topic, messageType, service string, value map[string]string) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType(messageType).
		ExpectService(service).
		ExpectStringMap(value)
}

func ExpectIntMap(topic, messageType, service string, value map[string]int64) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType(messageType).
		ExpectService(service).
		ExpectIntMap(value)
}

func ExpectFloatMap(topic, messageType, service string, value map[string]float64) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType(messageType).
		ExpectService(service).
		ExpectFloatMap(value)
}

func ExpectBoolMap(topic, messageType, service string, value map[string]bool) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType(messageType).
		ExpectService(service).
		ExpectBoolMap(value)
}

func ExpectStringArray(topic, messageType, service string, value []string) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType(messageType).
		ExpectService(service).
		ExpectStringArray(value)
}

func ExpectIntArray(topic, messageType, service string, value []int64) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType(messageType).
		ExpectService(service).
		ExpectIntArray(value)
}

func ExpectFloatArray(topic, messageType, service string, value []float64) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType(messageType).
		ExpectService(service).
		ExpectFloatArray(value)
}

func ExpectBoolArray(topic, messageType, service string, value []bool) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType(messageType).
		ExpectService(service).
		ExpectBoolArray(value)
}

func ExpectError(topic, service string) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType(router.EvtErrorReport).
		ExpectService(service)
}

func ExpectSuccess(topic, service string) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType(router.EvtSuccessReport).
		ExpectService(service)
}

func NewExpectation(voters ...router.MessageVoter) *Expectation {
	return &Expectation{
		Voters:     voters,
		Occurrence: AtLeastOnce,
	}
}

type Expectation struct {
	Voters     []router.MessageVoter
	Reply      *fimpgo.FimpMessage
	ReplyFn    func() *fimpgo.FimpMessage
	Publish    *fimpgo.Message
	PublishFn  func() *fimpgo.Message
	Occurrence Occurrence

	called int
}

func (e *Expectation) Expect(voters ...router.MessageVoter) *Expectation {
	e.Voters = append(e.Voters, voters...)

	return e
}

func (e *Expectation) ExpectTopic(topic string) *Expectation {
	e.Voters = append(e.Voters, router.ForTopic(topic))

	return e
}

func (e *Expectation) ExpectService(service string) *Expectation {
	e.Voters = append(e.Voters, router.ForService(service))

	return e
}

func (e *Expectation) ExpectType(messageType string) *Expectation {
	e.Voters = append(e.Voters, router.ForType(messageType))

	return e
}

func (e *Expectation) ExpectString(value string) *Expectation {
	e.Voters = append(e.Voters, router.MessageVoterFn(func(message *fimpgo.Message) bool {
		v, err := message.Payload.GetStringValue()
		if err != nil {
			return false
		}

		return v == value
	}))

	return e
}

func (e *Expectation) ExpectBool(value bool) *Expectation {
	e.Voters = append(e.Voters, router.MessageVoterFn(func(message *fimpgo.Message) bool {
		v, err := message.Payload.GetBoolValue()
		if err != nil {
			return false
		}

		return v == value
	}))

	return e
}

func (e *Expectation) ExpectInt(value int64) *Expectation {
	e.Voters = append(e.Voters, router.MessageVoterFn(func(message *fimpgo.Message) bool {
		v, err := message.Payload.GetIntValue()
		if err != nil {
			return false
		}

		return v == value
	}))

	return e
}

func (e *Expectation) ExpectFloat(value float64) *Expectation {
	e.Voters = append(e.Voters, router.MessageVoterFn(func(message *fimpgo.Message) bool {
		v, err := message.Payload.GetFloatValue()
		if err != nil {
			return false
		}

		return v == value
	}))

	return e
}

func (e *Expectation) ExpectObject(object interface{}) *Expectation {
	e.Voters = append(e.Voters, router.MessageVoterFn(func(message *fimpgo.Message) bool {
		raw, err := json.Marshal(object)
		if err != nil {
			return false
		}

		return cmp.Equal(raw, message.Payload.GetRawObjectValue())
	}))

	return e
}

func (e *Expectation) ExpectNull() *Expectation {
	e.Voters = append(e.Voters, router.MessageVoterFn(func(message *fimpgo.Message) bool {
		if message.Payload.ValueType != fimpgo.VTypeNull {
			return false
		}

		return message.Payload.Value == nil
	}))

	return e
}

func (e *Expectation) ExpectStringMap(value map[string]string) *Expectation {
	e.Voters = append(e.Voters, router.MessageVoterFn(func(message *fimpgo.Message) bool {
		v, err := message.Payload.GetStrMapValue()
		if err != nil {
			return false
		}

		return cmp.Equal(value, v)
	}))

	return e
}

func (e *Expectation) ExpectFloatMap(value map[string]float64) *Expectation {
	e.Voters = append(e.Voters, router.MessageVoterFn(func(message *fimpgo.Message) bool {
		v, err := message.Payload.GetFloatMapValue()
		if err != nil {
			return false
		}

		return cmp.Equal(value, v)
	}))

	return e
}

func (e *Expectation) ExpectIntMap(value map[string]int64) *Expectation {
	e.Voters = append(e.Voters, router.MessageVoterFn(func(message *fimpgo.Message) bool {
		v, err := message.Payload.GetIntMapValue()
		if err != nil {
			return false
		}

		return cmp.Equal(value, v)
	}))

	return e
}

func (e *Expectation) ExpectBoolMap(value map[string]bool) *Expectation {
	e.Voters = append(e.Voters, router.MessageVoterFn(func(message *fimpgo.Message) bool {
		v, err := message.Payload.GetBoolMapValue()
		if err != nil {
			return false
		}

		return cmp.Equal(value, v)
	}))

	return e
}

func (e *Expectation) ExpectStringArray(value []string) *Expectation {
	e.Voters = append(e.Voters, router.MessageVoterFn(func(message *fimpgo.Message) bool {
		v, err := message.Payload.GetStrArrayValue()
		if err != nil {
			return false
		}

		return cmp.Equal(value, v)
	}))

	return e
}

func (e *Expectation) ExpectFloatArray(value []float64) *Expectation {
	e.Voters = append(e.Voters, router.MessageVoterFn(func(message *fimpgo.Message) bool {
		v, err := message.Payload.GetFloatArrayValue()
		if err != nil {
			return false
		}

		return cmp.Equal(value, v)
	}))

	return e
}

func (e *Expectation) ExpectIntArray(value []int64) *Expectation {
	e.Voters = append(e.Voters, router.MessageVoterFn(func(message *fimpgo.Message) bool {
		v, err := message.Payload.GetIntArrayValue()
		if err != nil {
			return false
		}

		return cmp.Equal(value, v)
	}))

	return e
}

func (e *Expectation) ExpectBoolArray(value []bool) *Expectation {
	e.Voters = append(e.Voters, router.MessageVoterFn(func(message *fimpgo.Message) bool {
		v, err := message.Payload.GetBoolArrayValue()
		if err != nil {
			return false
		}

		return cmp.Equal(value, v)
	}))

	return e
}

func (e *Expectation) ExpectProperty(propertyName string, propertyValue interface{}) *Expectation {
	e.Voters = append(e.Voters, router.MessageVoterFn(func(message *fimpgo.Message) bool {
		property, ok := message.Payload.Properties[propertyName]
		if !ok {
			return false
		}

		return cmp.Equal(property, propertyValue)
	}))

	return e
}

func (e *Expectation) ExpectNoProperty(propertyName string) *Expectation {
	e.Voters = append(e.Voters, router.MessageVoterFn(func(message *fimpgo.Message) bool {
		_, ok := message.Payload.Properties[propertyName]
		return !ok
	}))

	return e
}

func (e *Expectation) ReplyWith(reply *fimpgo.FimpMessage) *Expectation {
	e.Reply = reply

	return e
}

func (e *Expectation) ReplyWithFn(replyFn func() *fimpgo.FimpMessage) *Expectation {
	e.ReplyFn = replyFn

	return e
}

func (e *Expectation) PublishInResponse(publish *fimpgo.Message) *Expectation {
	e.Publish = publish

	return e
}

func (e *Expectation) PublishInResponseFn(publishFn func() *fimpgo.Message) *Expectation {
	e.PublishFn = publishFn

	return e
}

func (e *Expectation) AtLeastOnce() *Expectation {
	e.Occurrence = AtLeastOnce

	return e
}

func (e *Expectation) ExactlyOnce() *Expectation {
	e.Occurrence = ExactlyOnce

	return e
}

func (e *Expectation) AtMostOnce() *Expectation {
	e.Occurrence = AtMostOnce

	return e
}

func (e *Expectation) Never() *Expectation {
	e.Occurrence = Never

	return e
}

func (e *Expectation) vote(message *fimpgo.Message) (voted bool, votesCount int) {
	voted = true

	for _, v := range e.Voters {
		if !v.Vote(message) {
			voted = false

			continue
		}

		votesCount++
	}

	return
}

func (e *Expectation) assert() bool {
	switch e.Occurrence {
	case AtLeastOnce:
		return e.called >= 1
	case ExactlyOnce:
		return e.called == 1
	case AtMostOnce:
		return e.called <= 1
	case Never:
		return e.called == 0
	}

	return false
}
