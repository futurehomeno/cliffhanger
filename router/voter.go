package router

import (
	"github.com/futurehomeno/fimpgo"
)

// MessageVoter is an interface representing a message voter for a particular routing.
type MessageVoter interface {
	// Vote provides with a binary answer whether the message should be handled by a particular routing.
	Vote(msg *fimpgo.Message) bool
}

// MessageVoterFn is an adapter allowing usage of anonymous function as a service meeting message voter interface.
type MessageVoterFn func(msg *fimpgo.Message) bool

// Vote provides with a binary answer whether the message should be handled by a particular routing.
func (f MessageVoterFn) Vote(msg *fimpgo.Message) bool {
	return f(msg)
}

// ForService is a message voter allowing a routing to handle message only if it is relevant.
func ForService(service string) MessageVoter {
	return MessageVoterFn(func(msg *fimpgo.Message) bool {
		return msg.Payload.Service == service
	})
}

// ForType is a message voter allowing a routing to handle message only if it is relevant.
func ForType(messageType string) MessageVoter {
	return MessageVoterFn(func(msg *fimpgo.Message) bool {
		return msg.Payload.Type == messageType
	})
}

// ForServiceAndType is a message voter allowing a routing to handle message only if it is relevant.
func ForServiceAndType(service, messageType string) MessageVoter {
	serviceVoter := ForService(service)
	typeVoter := ForType(messageType)

	return MessageVoterFn(func(msg *fimpgo.Message) bool {
		return serviceVoter.Vote(msg) && typeVoter.Vote(msg)
	})
}
