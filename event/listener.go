package event

import (
	"fmt"
	"sync"
)

type Processor interface {
	Process(event *Event)
}

type ProcessorFn func(event *Event)

func (p ProcessorFn) Process(event *Event) {
	p(event)
}

type Listener interface {
	Start() error
	Stop() error
}

func NewListener(processor Processor, manager Manager, subID string, buffer int, filters ...Filter) Listener {
	return &listener{
		processor: processor,
		manager:   manager,
		subID:     subID,
		buffer:    buffer,
		filters:   filters,
		lock:      &sync.Mutex{},
		waitGroup: &sync.WaitGroup{},
	}
}

type listener struct {
	processor Processor
	manager   Manager

	subID   string
	buffer  int
	filters []Filter

	eventCh   chan *Event
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

	l.eventCh = l.manager.Subscribe(l.subID, l.buffer, l.filters...)
	l.closeCh = make(chan struct{})

	l.waitGroup.Add(1)

	go l.process()

	return nil
}

func (l *listener) process() {
	defer l.waitGroup.Done()

	for {
		select {
		case event := <-l.eventCh:
			l.processor.Process(event)

		case <-l.closeCh:
			return
		}
	}
}

func (l *listener) Stop() error {
	l.lock.Lock()
	defer l.lock.Unlock()

	if l.closeCh == nil {
		return fmt.Errorf("listener: already stopped")
	}

	l.manager.Unsubscribe(l.subID)

	close(l.closeCh)

	l.waitGroup.Wait()

	l.closeCh = nil
	l.eventCh = nil

	return nil
}
