package config

import (
	"github.com/futurehomeno/cliffhanger/event"
)

// Constants defining event domain and classes.
const (
	eventDomain                   = "config"
	eventClassConfigurationChange = "configuration_change"
)

// NewConfigurationChangeEvent creates a new schedule update event.
func NewConfigurationChangeEvent(service, setting string) event.Event {
	return event.NewWithPayload(eventDomain, eventClassConfigurationChange, &configurationChange{
		Service: service,
		Setting: setting,
	})
}

// configurationChange represents a configuration change event.
type configurationChange struct {
	Service string
	Setting string
}

// WaitForConfigurationUpdate creates a filter for configuration change events.
func WaitForConfigurationUpdate(service, setting string) event.Filter {
	return event.And(
		event.WaitForDomain(eventDomain),
		event.WaitForClass(eventClassConfigurationChange),
		event.WaitForPayload(&configurationChange{
			Service: service,
			Setting: setting,
		}),
	)
}
