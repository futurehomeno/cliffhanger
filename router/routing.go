package router

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
)

// TopicPatternAdapter returns a topic pattern for an adapter useful for subscriptions.
func TopicPatternAdapter(serviceName string) string {
	return fmt.Sprintf("pt:j1/+/rt:ad/rn:%s/ad:1", serviceName)
}

// TopicPatternDevices returns a topic pattern for devices useful for subscriptions.
func TopicPatternDevices(serviceName string) string {
	return fmt.Sprintf("pt:j1/+/rt:dev/rn:%s/ad:1/#", serviceName)
}

// TopicPatternApplication returns a topic pattern for application useful for subscriptions.
func TopicPatternApplication(serviceName string) string {
	return fmt.Sprintf("pt:j1/+/rt:app/rn:%s/ad:1", serviceName)
}

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
func (r *Routing) vote(msg *fimpgo.Message) bool {
	for _, v := range r.voters {
		if !v.Vote(msg) {
			return false
		}
	}

	return true
}