package adapter

import (
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/event"
)

const (
	EventDomainAdapterService = "adapter_service"
	EventDomainAdapterThing   = "adapter_thing"

	EventClassAdapterThing        = "thing"
	EventClassInclusionReportSent = "inclusion_report_sent"
)

type (
	ServiceEvent interface {
		event.Event

		ServiceName() fimptype.ServiceNameT
		Address() string
		HasChanged() bool

		setEvent(event event.Event)
		setServiceName(serviceName fimptype.ServiceNameT)
		setAddress(address string)
	}

	ThingEvent interface {
		event.Event

		Address() string
	}

	serviceEvent struct {
		event.Event

		serviceName fimptype.ServiceNameT
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

func NewThingEvent(address string, payload any) ThingEvent {
	return &thingEvent{
		Event:   event.NewWithPayload(EventDomainAdapterThing, EventClassAdapterThing, payload),
		address: address,
	}
}

func NewInclusionReportSentEvent(address string, payload fimptype.ThingInclusionReport) ThingEvent {
	return &thingEvent{
		Event:   event.NewWithPayload(EventDomainAdapterThing, EventClassInclusionReportSent, payload),
		address: address,
	}
}

func (e *serviceEvent) ServiceName() fimptype.ServiceNameT {
	return e.serviceName
}

func (e *serviceEvent) Address() string {
	return e.address
}

func (e *serviceEvent) HasChanged() bool {
	return e.hasChanged
}

func (e *serviceEvent) EventType() string {
	return e.eventType
}

func (e *serviceEvent) setEvent(event event.Event) {
	e.Event = event
}

func (e *serviceEvent) setServiceName(serviceName fimptype.ServiceNameT) {
	e.serviceName = serviceName
}

func (e *serviceEvent) setAddress(address string) {
	e.address = address
}

func (e *thingEvent) Address() string {
	return e.address
}

func WaitForServiceEvent() event.Filter {
	return event.WaitFor[ServiceEvent]()
}

func WaitForThingEvent() event.Filter {
	return event.WaitFor[ThingEvent]()
}
