package event

import (
	"github.com/google/go-cmp/cmp"
)

type Event interface {
	Domain() string
	Class() string
	Payload() interface{}
}

func New(domain, class string) Event {
	return &event{
		domain: domain,
		class:  class,
	}
}

type event struct {
	domain string
	class  string

	payload interface{}
}

func (e *event) Domain() string {
	return e.domain
}

func (e *event) Class() string {
	return e.class
}

func (e *event) Payload() interface{} { return e.payload }

func NewWithPayload(domain, class string, payload interface{}) Event {
	return &event{
		domain:  domain,
		class:   class,
		payload: payload,
	}
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
		return event.Class() == class
	})
}

func WaitForPayload(payload interface{}) Filter {
	return FilterFn(func(e Event) bool {
		if e.Payload() == nil {
			return false
		}

		return cmp.Equal(e.Payload(), payload)
	})
}
