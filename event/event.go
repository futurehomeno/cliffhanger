package event

import (
	"github.com/google/go-cmp/cmp"
)

type Event struct {
	Domain  string
	Payload interface{}
}

func New(domain string, payload interface{}) *Event {
	return &Event{
		Domain:  domain,
		Payload: payload,
	}
}

type Filter interface {
	Filter(event *Event) bool
}

type FilterFn func(event *Event) bool

func (f FilterFn) Filter(event *Event) bool {
	return f(event)
}

func Or(filter ...Filter) Filter {
	return FilterFn(func(event *Event) bool {
		for _, f := range filter {
			if f.Filter(event) {
				return true
			}
		}

		return false
	})
}

func And(filter ...Filter) Filter {
	return FilterFn(func(event *Event) bool {
		for _, f := range filter {
			if !f.Filter(event) {
				return false
			}
		}

		return true
	})
}

func WaitForDomain(domain string) Filter {
	return FilterFn(func(event *Event) bool {
		return event.Domain == domain
	})
}

func WaitForPayload(payload interface{}) Filter {
	return FilterFn(func(event *Event) bool {
		return cmp.Equal(event.Payload, payload)
	})
}
