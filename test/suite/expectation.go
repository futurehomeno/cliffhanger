package suite

import (
	"encoding/json"

	"github.com/futurehomeno/fimpgo"
	"github.com/google/go-cmp/cmp"

	"github.com/futurehomeno/cliffhanger/router"
)

type Occurrence int

const (
	AtLeastOnce Occurrence = iota
	ExactlyOnce
	AtMostOnce
)

func ExpectString(topic, messageType, service, value string) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType(messageType).
		ExpectService(service).
		ExpectString(value)
}

func ExpectObject(topic, messageType, service string, object interface{}) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType(messageType).
		ExpectService(service).
		ExpectObject(object)
}

func ExpectError(topic, service string) *Expectation {
	return NewExpectation().
		ExpectTopic(topic).
		ExpectType("evt.error.report").
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
	Publish    *fimpgo.Message
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

func (e *Expectation) ReplyWith(reply *fimpgo.FimpMessage) *Expectation {
	e.Reply = reply

	return e
}

func (e *Expectation) PublishInResponse(message *fimpgo.Message) *Expectation {
	e.Publish = message

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

func (e *Expectation) vote(message *fimpgo.Message) bool {
	for _, v := range e.Voters {
		if !v.Vote(message) {
			return false
		}
	}

	return true
}

func (e *Expectation) assert() bool {
	switch e.Occurrence {
	case AtLeastOnce:
		return e.called >= 1
	case ExactlyOnce:
		return e.called == 1
	case AtMostOnce:
		return e.called <= 1
	}

	return false
}
