package router

import (
	"github.com/futurehomeno/fimpgo"
)

// MessageHandler is an interface representing a message handler service.
type MessageHandler interface {
	// Handle handles the incoming message and optionally returns a response. If no response is expected a nil message should be returned.
	Handle(message *fimpgo.Message) (reply *fimpgo.Message)
}

// MessageHandlerFn is an adapter allowing usage of anonymous function as a service meeting message handler interface.
type MessageHandlerFn func(message *fimpgo.Message) (reply *fimpgo.Message)

// Handle handles the incoming message and optionally returns a response. If no response is expected a nil message should be returned.
func (f MessageHandlerFn) Handle(message *fimpgo.Message) (reply *fimpgo.Message) {
	return f(message)
}
