package router

import (
	"github.com/futurehomeno/fimpgo"
)

// Constant defining error type of message for error responses and its properties.
const (
	EvtSuccessReport = "evt.success.report"
	EvtErrorReport   = "evt.error.report"

	PropertyMsg        = "msg"
	PropertyCmdTopic   = "cmd_topic"
	PropertyCmdService = "cmd_service"
	PropertyCmdType    = "cmd_type"
)

// Routing is an object representing a particular routing. It contains a message handler and a set of message voters.
type Routing struct {
	handler MessageHandler
	voters  []MessageVoter
}

// NewRouting creates a new routing from provided message handler and message voters.
func NewRouting(handler MessageHandler, voters ...MessageVoter) *Routing {
	return &Routing{
		handler: handler,
		voters:  voters,
	}
}

// Wrap is a helper which creates a new routing with additional message voters.
func (r *Routing) Wrap(voters ...MessageVoter) *Routing {
	return NewRouting(r.handler, append(voters, r.voters...)...)
}

// vote checks if all set conditions are met by executing all registered voters.
func (r *Routing) vote(message *fimpgo.Message) bool {
	for _, v := range r.voters {
		if !v.Vote(message) {
			return false
		}
	}

	return true
}

// Combine is a helper to easily combine multiple instances or slices of routing into one slice.
func Combine[T []*Routing | *Routing](parts ...T) []*Routing {
	var combined []*Routing

	for _, p := range parts {
		switch p := any(p).(type) {
		case *Routing:
			combined = append(combined, p)
		case []*Routing:
			combined = append(combined, p...)
		}
	}

	return combined
}
