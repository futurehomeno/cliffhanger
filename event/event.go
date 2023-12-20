package event

import (
	"github.com/google/go-cmp/cmp"
)

type Event interface {
	Domain() string
	Class() string
}

func New(domain string, class string) Event {
	return &event{
		domain: domain,
		class:  class,
	}
}

type event struct {
	domain string
	class  string
}

func (e *event) Domain() string {
	return e.domain
}

func (e *event) Class() string {
	return e.class

}

func NewWithPayload(domain string, class string, payload interface{}) Event {
	return &eventWithPayload{
		Event:   New(domain, class),
		payload: payload,
	}
}

type eventWithPayload struct {
	Event

	payload interface{}
}

type Filter interface {
	Filter(event Event) bool
}

type FilterFn func(event Event) bool

func (f FilterFn) Filter(event Event) bool {
	return f(event)
}

func Or(filter ...Filter) Filter {
	return FilterFn(func(event Event) bool {
		for _, f := range filter {
			if f.Filter(event) {
				return true
			}
		}

		return false
	})
}

func And(filter ...Filter) Filter {
	return FilterFn(func(event Event) bool {
		for _, f := range filter {
			if !f.Filter(event) {
				return false
			}
		}

		return true
	})
}

func WaitForDomain(domain string) Filter {
	return FilterFn(func(event Event) bool {
		return event.Domain() == domain
	})
}

func WaitForClass(class string) Filter {
	return FilterFn(func(event Event) bool {
		return event.Domain() == class
	})
}

func WaitForPayload(payload interface{}) Filter {
	return FilterFn(func(event Event) bool {
		e, ok := event.(*eventWithPayload)
		if !ok {
			return false
		}

		return cmp.Equal(e.payload, payload)
	})
}
