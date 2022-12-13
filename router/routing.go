package router

import (
	"github.com/futurehomeno/fimpgo"
)

// EvtErrorReport is a type of message for error responses.
const EvtErrorReport = "evt.error.report"

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

// vote checks if all set conditions are met by executing all registered voters.
func (r *Routing) vote(message *fimpgo.Message) bool {
	for _, v := range r.voters {
		if !v.Vote(message) {
			return false
		}
	}

	return true
}

// Combine is a helper to easily combine multiple slices of routing into one.
func Combine(parts ...[]*Routing) []*Routing {
	var combined []*Routing

	for _, p := range parts {
		combined = append(combined, p...)
	}

	return combined
}
