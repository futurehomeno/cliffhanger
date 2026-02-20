package router

import (
	"fmt"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
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

// MessageProcessor is an interface representing a message processor service.
type MessageProcessor interface {
	// Process is responsible for processing incoming message and returning response payload and optionally an error.
	Process(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error)
}

// MessageProcessorFn is an adapter allowing usage of anonymous function as a service meeting message processor interface.
type MessageProcessorFn func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error)

// Process is responsible for processing incoming message and returning response payload and optionally an error.
func (f MessageProcessorFn) Process(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
	return f(message)
}

// MessageHandlerLocker is an interface representing a locker used to prevent concurrent message processing.
type MessageHandlerLocker interface {
	// Lock tries to lock message processing. Returns true if lock was successful and false if it was locked previously.
	Lock() bool
	// Unlock unlocks the message processing.
	Unlock()
}

// NewMessageHandlerLocker creates a new instance of a message handler locker service.
func NewMessageHandlerLocker() MessageHandlerLocker {
	return &messageProcessorLocker{
		lock: &sync.Mutex{},
	}
}

// messageProcessorLocker is a private implementation of a message handler locker interface.
type messageProcessorLocker struct {
	lock     *sync.Mutex
	isLocked bool
}

// Lock tries to lock message processing. Returns true if lock was successful and false if it was locked previously.
func (m *messageProcessorLocker) Lock() bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.isLocked {
		return false
	}

	m.isLocked = true

	return true
}

// Unlock unlocks the message processing.
func (m *messageProcessorLocker) Unlock() {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.isLocked = false
}

// NewMessageHandler creates new instance of a message handler with a set of useful default behaviors.
// - handler will infer a default response address from request message, unless this behavior is overridden by WithDefaultAddress option.
// - on error handler will respond with error message, unless this behavior is overridden by WithSilentErrors option.
func NewMessageHandler(processor MessageProcessor, options ...MessageHandlerOption) MessageHandler {
	h := &messageHandler{
		processor: processor,
	}

	for _, o := range options {
		o.apply(h)
	}

	return h
}

// messageHandler is a private implementation of a message handler interface.
type messageHandler struct {
	processor MessageProcessor

	defaultAddress *fimpgo.Address
	silentErrors   bool
	confirmSuccess bool
	locker         MessageHandlerLocker
}

// Handle handles the incoming message and optionally returns a response.
func (m *messageHandler) Handle(message *fimpgo.Message) *fimpgo.Message {
	if m.locker != nil {
		if !m.locker.Lock() {
			return m.handleError(
				message,
				fmt.Errorf("another operation is already running, skipping message"))
		}
		defer m.locker.Unlock()
	}

	reply, err := m.processor.Process(message)
	if err != nil {
		return m.handleError(message, err)
	}

	return m.handleReply(message, reply)
}

// handleReply returns reply message with an address.
func (m *messageHandler) handleReply(requestMessage *fimpgo.Message, reply *fimpgo.FimpMessage) *fimpgo.Message {
	if reply == nil {
		if !m.confirmSuccess {
			return nil
		}

		return &fimpgo.Message{
			Addr: m.getResponseAddress(requestMessage.Addr),
			Payload: fimpgo.NewMessage(
				EvtSuccessReport,
				requestMessage.Payload.Service,
				fimptype.VTypeNull,
				nil,
				map[string]string{
					PropertyCmdTopic:   requestMessage.Topic,
					PropertyCmdService: requestMessage.Payload.Service.Str(),
					PropertyCmdType:    requestMessage.Payload.Interface,
				},
				nil,
				requestMessage.Payload,
			),
		}
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
		WithField("type", requestMessage.Payload.Interface).
		Error("Process incoming msg")

	if m.silentErrors {
		return nil
	}

	reply := &fimpgo.Message{
		Addr: m.getResponseAddress(requestMessage.Addr),
		Payload: fimpgo.NewMessage(
			EvtErrorReport,
			requestMessage.Payload.Service,
			fimptype.VTypeString,
			"failed to process incoming message",
			map[string]string{
				PropertyMsg:        err.Error(),
				PropertyCmdTopic:   requestMessage.Topic,
				PropertyCmdService: requestMessage.Payload.Service.Str(),
				PropertyCmdType:    requestMessage.Payload.Interface,
			},
			nil,
			requestMessage.Payload,
		),
	}

	// Do not store device errors in the storage.
	if reply.Addr.ResourceType == fimptype.ResourceTypeDevice && reply.Payload.Storage == nil {
		reply.Payload.WithStorageStrategy(fimpgo.StorageStrategySkip, "")
	}

	return reply
}

// getResponseAddress returns response address.
func (m *messageHandler) getResponseAddress(requestAddress *fimpgo.Address) *fimpgo.Address {
	a := requestAddress
	if m.defaultAddress != nil {
		a = m.defaultAddress
	}

	return &fimpgo.Address{
		PayloadType:     a.PayloadType,
		MsgType:         fimptype.MsgTypeEvt,
		ResourceType:    a.ResourceType,
		ResourceName:    a.ResourceName,
		ResourceAddress: a.ResourceAddress,
		ServiceName:     a.ServiceName,
		ServiceAddress:  a.ServiceAddress,
	}
}

// MessageHandlerOption is an interface representing a message handler configuration option.
type MessageHandlerOption interface {
	// apply applies option to the message handler.
	apply(h *messageHandler)
}

// messageHandlerOptionFn is an adapter allowing usage of anonymous function as a service meeting message handler option interface.
type messageHandlerOptionFn func(h *messageHandler)

// apply applies option to the message handler.
func (f messageHandlerOptionFn) apply(h *messageHandler) {
	f(h)
}

// WithSilentErrors makes handler only log errors and not respond with error messages.
func WithSilentErrors() MessageHandlerOption {
	return messageHandlerOptionFn(func(h *messageHandler) {
		h.silentErrors = true
	})
}

// WithDefaultAddress makes handler use a provided address, instead of inferring a response address out of request message address.
func WithDefaultAddress(defaultAddress *fimpgo.Address) MessageHandlerOption {
	return messageHandlerOptionFn(func(h *messageHandler) {
		h.defaultAddress = defaultAddress
	})
}

// WithLock makes sure handler will process only one message at a time, ignoring other ones.
func WithLock() MessageHandlerOption {
	return messageHandlerOptionFn(func(h *messageHandler) {
		h.locker = NewMessageHandlerLocker()
	})
}

// WithExternalLock makes sure handler will process message only if an external lock allows for it.
func WithExternalLock(locker MessageHandlerLocker) MessageHandlerOption {
	return messageHandlerOptionFn(func(h *messageHandler) {
		h.locker = locker
	})
}

// WithSuccessConfirmation makes handler respond with success report if message processor do not reply with a message.
func WithSuccessConfirmation() MessageHandlerOption {
	return messageHandlerOptionFn(func(h *messageHandler) {
		h.confirmSuccess = true
	})
}
