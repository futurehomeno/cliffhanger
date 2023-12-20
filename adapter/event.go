package adapter

import (
	"github.com/futurehomeno/cliffhanger/event"
)

const (
	EventDomainAdapterService = "adapter_service"
)

func NewServiceEvent(eventType string, hasChanged bool) ServiceEvent {
	return &serviceEvent{
		eventType:  eventType,
		hasChanged: hasChanged,
	}
}

type ServiceEvent interface {
	event.Event

	ServiceName() string
	Address() string
	EventType() string
	HasChanged() bool

	setEvent(event event.Event) ServiceEvent
	setServiceName(serviceName string) ServiceEvent
	setAddress(address string) ServiceEvent
}

type serviceEvent struct {
	event.Event

	serviceName string
	address     string
	eventType   string
	hasChanged  bool
}

func (e *serviceEvent) ServiceName() string {
	return e.serviceName
}

func (e *serviceEvent) Address() string {
	return e.address
}

func (e *serviceEvent) EventType() string {
	return e.eventType
}

func (e *serviceEvent) HasChanged() bool {
	return e.hasChanged
}

func (e *serviceEvent) setEvent(event event.Event) ServiceEvent {
	e.Event = event

	return e
}

func (e *serviceEvent) setServiceName(serviceName string) ServiceEvent {
	e.serviceName = serviceName

	return e
}

func (e *serviceEvent) setAddress(address string) ServiceEvent {
	e.address = address

	return e
}
