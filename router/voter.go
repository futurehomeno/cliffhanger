package router

import (
	"strings"

	"github.com/futurehomeno/fimpgo"
)

// MessageVoter is an interface representing a message voter for a particular routing.
type MessageVoter interface {
	// Vote provides with a binary answer whether the message should be handled by a particular routing.
	Vote(message *fimpgo.Message) bool
}

// MessageVoterFn is an adapter allowing usage of anonymous function as a service meeting message voter interface.
type MessageVoterFn func(message *fimpgo.Message) bool

// Vote provides with a binary answer whether the message should be handled by a particular routing.
func (f MessageVoterFn) Vote(message *fimpgo.Message) bool {
	return f(message)
}

// Or is a composite voter returning true if at least one of voters returned true, equal to logic OR statement.
func Or(voters ...MessageVoter) MessageVoter {
	return MessageVoterFn(func(message *fimpgo.Message) bool {
		for _, v := range voters {
			if v.Vote(message) {
				return true
			}
		}

		return false
	})
}

// ForTopic is a message voter allowing a routing to handle message only if it is relevant.
func ForTopic(topic string) MessageVoter {
	return MessageVoterFn(func(message *fimpgo.Message) bool {
		return message.Topic == topic
	})
}

// ForResourceType is a message voter allowing a routing to handle message only if it is relevant.
func ForResourceType(resourceType string) MessageVoter {
	return MessageVoterFn(func(message *fimpgo.Message) bool {
		return strings.Contains(message.Topic, "/rt:"+resourceType+"/")
	})
}

// ForService is a message voter allowing a routing to handle message only if it is relevant.
func ForService(service string) MessageVoter {
	return MessageVoterFn(func(message *fimpgo.Message) bool {
		return message.Payload.Service == service
	})
}

// ForServicePrefix is a message voter allowing a routing to handle message only if it is relevant.
func ForServicePrefix(prefix string) MessageVoter {
	return MessageVoterFn(func(message *fimpgo.Message) bool {
		return strings.HasPrefix(message.Payload.Service, prefix)
	})
}

// ForType is a message voter allowing a routing to handle message only if it is relevant.
func ForType(messageType string) MessageVoter {
	return MessageVoterFn(func(message *fimpgo.Message) bool {
		return message.Payload.Type == messageType
	})
}

// ForServiceAndType is a message voter allowing a routing to handle message only if it is relevant.
func ForServiceAndType(service, messageType string) MessageVoter {
	serviceVoter := ForService(service)
	typeVoter := ForType(messageType)

	return MessageVoterFn(func(message *fimpgo.Message) bool {
		return serviceVoter.Vote(message) && typeVoter.Vote(message)
	})
}
