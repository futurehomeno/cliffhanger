package router

import (
	"fmt"
	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	"strings"
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

// ErrorableHandlerFn represents a message handler that can return an error during processing.
type ErrorableHandlerFn func(message *fimpgo.Message) (reply *fimpgo.Message, err error)

// ErrorMessageHandler is a handler that can process errorable function and provides a generic way of error handling for FIMP handlers.
func ErrorMessageHandler(h ErrorableHandlerFn) MessageHandler {
	return MessageHandlerFn(func(message *fimpgo.Message) (reply *fimpgo.Message) {
		reply, err := h(message)
		if err != nil {
			return errMsg(err, message)
		}

		return
	})
}

// DefaultAdressHandler is a handler that ensures the response address is defined on a message.
func DefaultAdressHandler(h MessageHandlerFn) MessageHandler {
	return MessageHandlerFn(func(message *fimpgo.Message) (reply *fimpgo.Message) {
		reply = h(message)

		if reply.Addr != nil {
			return
		}

		reply.Addr = message.Addr
		reply.Addr.MsgType = "evt"

		return
	})
}

func errMsg(err error, message *fimpgo.Message) *fimpgo.Message {
	ensureResponseTopic(message)

	message.Payload.Type = modifyMsgType(message.Payload.Type)
	message.Payload.ValueType = fimpgo.VTypeString
	message.Payload.Value = err.Error()

	return message
}

// ensureResponseTopic makes sure the response topic is defined.
// If the address or response topic are empty, the original topic is used (with some required modifications).
func ensureResponseTopic(message *fimpgo.Message) {
	if message.Payload.ResponseToTopic != "" {
		return
	}

	if message.Addr == nil {
		addr, err := fimpgo.NewAddressFromString(message.Topic)
		if err != nil {
			log.
				WithError(err).
				WithField("topic", message.Topic).
				Error("cannot parse a topic as fimp address")
		}

		message.Addr = addr
	}

	message.Addr.MsgType = "evt"
}

func modifyMsgType(t string) string {
	s := strings.Split(t, ".")
	if len(s) < 3 {
		return "evt.error"
	}

	return fmt.Sprintf("evt.%s.error", s[1])
}
