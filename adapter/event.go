package adapter

import (
	"github.com/futurehomeno/cliffhanger/event"
)

const (
	EventDomainAdapterService = "adapter_service"
	EventDomainAdapterThing   = "adapter_thing"

	EventClassAdapterThing = "thing"
)

type (
	ServiceEvent interface {
		event.Event

		ServiceName() string
		Address() string
		HasChanged() bool

		setEvent(event event.Event)
		setServiceName(serviceName string)
		setAddress(address string)
	}

	ThingEvent interface {
		event.Event

		Address() string
	}

	serviceEvent struct {
		event.Event

		serviceName string
		address     string
		eventType   string
		hasChanged  bool
	}

	thingEvent struct {
		event.Event

		address string
	}
)

func NewServiceEvent(eventType string, hasChanged bool) ServiceEvent {
	return &serviceEvent{
		eventType:  eventType,
		hasChanged: hasChanged,
	}
}

func NewThingEvent(address string, payload interface{}) ThingEvent {
	return &thingEvent{
		Event:   event.NewWithPayload(EventDomainAdapterThing, EventClassAdapterThing, payload),
		address: address,
	}
}

func WaitForThingEvent() event.Filter {
	return event.FilterFn(func(e event.Event) bool {
		_, ok := e.(ThingEvent)

		return ok
	})
}

func (e *serviceEvent) ServiceName() string {
	return e.serviceName
}

func (e *serviceEvent) Address() string {
	return e.address
}

func (e *serviceEvent) HasChanged() bool {
	return e.hasChanged
}

func (e *serviceEvent) setEvent(event event.Event) {
	e.Event = event
}

func (e *serviceEvent) setServiceName(serviceName string) {
	e.serviceName = serviceName
}

func (e *serviceEvent) setAddress(address string) {
	e.address = address
}

func (e *thingEvent) Address() string {
	return e.address
}
