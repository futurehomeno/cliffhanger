package event

import (
	"fmt"
	"runtime/debug"
	"sync"

	log "github.com/sirupsen/logrus"
)

type Handler struct {
	processor Processor
	subID     string
	buffer    int
	filters   []Filter
	eventCh   chan Event
}

func NewHandler(processor Processor, subID string, buffer int, filters ...Filter) *Handler {
	return &Handler{
		processor: processor,
		subID:     subID,
		buffer:    buffer,
		filters:   filters,
	}
}

type Processor interface {
	Process(event Event)
}

type ProcessorFn func(event Event)

func (p ProcessorFn) Process(event Event) {
	p(event)
}

type Listener interface {
	Start() error
	Stop() error
}

func NewListener(manager Manager, handlers ...*Handler) Listener {
	return &listener{
		manager:   manager,
		lock:      &sync.Mutex{},
		waitGroup: &sync.WaitGroup{},
		handlers:  handlers,
	}
}

type listener struct {
	manager  Manager
	handlers []*Handler

	closeCh   chan struct{}
	lock      *sync.Mutex
	waitGroup *sync.WaitGroup
}

func (l *listener) Start() error {
	l.lock.Lock()
	defer l.lock.Unlock()

	if l.closeCh != nil {
		return fmt.Errorf("listener: already started")
	}

	l.closeCh = make(chan struct{})

	for _, h := range l.handlers {
		l.waitGroup.Add(1)

		h.eventCh = l.manager.Subscribe(h.subID, h.buffer, h.filters...)

		log.Infof("event listener: started listening for events with subscriber ID %s", h.subID)

		go l.startHandler(h)
	}

	return nil
}

func (l *listener) startHandler(h *Handler) {
	defer l.waitGroup.Done()

	for {
		select {
		case event := <-h.eventCh:
			l.doProcess(h.processor, event)

		case <-l.closeCh:
			return
		}
	}
}

// doProcess executes the event processor with a panic recovery.
func (l *listener) doProcess(processor Processor, event Event) {
	defer func() {
		if r := recover(); r != nil {
			log.WithField("stack", string(debug.Stack())).
				WithField("domain", event.Domain()).
				WithField("class", event.Class()).
				Errorf("event listener: panic occurred while processing the event: %+v", r)
		}
	}()

	processor.Process(event)
}

func (l *listener) Stop() error {
	l.lock.Lock()
	defer l.lock.Unlock()

	if l.closeCh == nil {
		return fmt.Errorf("listener: already stopped")
	}

	for _, h := range l.handlers {
		l.manager.Unsubscribe(h.subID)
	}

	close(l.closeCh)

	l.waitGroup.Wait()

	l.closeCh = nil

	return nil
}
