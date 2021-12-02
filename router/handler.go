package router

import (
	"fmt"
	"strings"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
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

// MessageProcessor is a type of a function responsible for processing incoming message and returning response payload and optionally an error.
type MessageProcessor func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error)

// NewMessageHandler creates new instance of a message handler.
func NewMessageHandler(processor MessageProcessor, options ...MessageHandlerOption) MessageHandler {
	h := &messageHandler{
		processor:    processor,
		silentErrors: false,
	}

	for _, o := range options {
		o.apply(h)
	}

	return h
}

// messageHandler is a private implementation of a message handler interface.
type messageHandler struct {
	processor      MessageProcessor
	defaultAddress *fimpgo.Address
	silentErrors   bool
}

// Handle handles the incoming message and optionally returns a response. If no response is expected a nil message should be returned.
func (m *messageHandler) Handle(message *fimpgo.Message) *fimpgo.Message {
	reply, err := m.processor(message)
	if err != nil {
		return m.handleError(message, err)
	}

	return m.handleReply(message, reply)
}

// handleReply returns reply message with an address.
func (m *messageHandler) handleReply(requestMessage *fimpgo.Message, reply *fimpgo.FimpMessage) *fimpgo.Message {
	if reply == nil {
		return nil
	}

	return &fimpgo.Message{
		Addr:    m.getResponseAddress(requestMessage.Addr),
		Payload: reply,
	}
}

// handleError handles message processing error.
func (m *messageHandler) handleError(requestMessage *fimpgo.Message, err error) *fimpgo.Message {
	log.
		WithError(err).
		WithField("topic", requestMessage.Topic).
		WithField("service", requestMessage.Payload.Service).
		WithField("type", requestMessage.Payload.Type).
		Error("handler failed to process incoming message")

	if m.silentErrors {
		return nil
	}

	return &fimpgo.Message{
		Addr: m.getResponseAddress(requestMessage.Addr),
		Payload: fimpgo.NewMessage(
			m.getErrorMessageType(requestMessage.Payload.Type),
			requestMessage.Payload.Service,
			fimpgo.VTypeString,
			err.Error(),
			nil,
			nil,
			requestMessage.Payload,
		),
	}
}

// getResponseAddress returns response address.
func (m *messageHandler) getResponseAddress(requestAddress *fimpgo.Address) *fimpgo.Address {
	a := requestAddress
	if m.defaultAddress != nil {
		a = m.defaultAddress
	}

	return &fimpgo.Address{
		PayloadType:     a.PayloadType,
		MsgType:         "evt",
		ResourceType:    a.ResourceType,
		ResourceName:    a.ResourceName,
		ResourceAddress: a.ResourceAddress,
		ServiceName:     a.ServiceName,
		ServiceAddress:  a.ServiceAddress,
	}
}

// getErrorMessageType returns error message type.
func (m *messageHandler) getErrorMessageType(messageType string) string {
	s := strings.Split(messageType, ".")
	if len(s) < 3 {
		return "evt.error"
	}

	return fmt.Sprintf("evt.%s.error", s[1])
}

// MessageHandlerOption is an interface representing a message handler configuration option.
type MessageHandlerOption interface {
	// apply applies option to the message handler.
	apply(h *messageHandler)
}

// messageHandlerOptionFn is an adapter allowing usage of anonymous function meeting message handler option interface.
type messageHandlerOptionFn func(h *messageHandler)

// apply applies option to the message handler.
func (f messageHandlerOptionFn) apply(h *messageHandler) {
	f(h)
}

func WithSilentErrors() MessageHandlerOption {
	return messageHandlerOptionFn(func(h *messageHandler) {
		h.silentErrors = true
	})
}

func WithDefaultAddress(defaultAddress *fimpgo.Address) MessageHandlerOption {
	return messageHandlerOptionFn(func(h *messageHandler) {
		h.defaultAddress = defaultAddress
	})
}
