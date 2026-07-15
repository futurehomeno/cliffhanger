package config

import (
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/event"
)

const (
	eventDomain                   = "config"
	eventClassConfigurationChange = "configuration_change"
)

func NewConfigurationChangeEvent(service fimptype.ServiceNameT, setting string) event.Event {
	return event.NewWithPayload(eventDomain, eventClassConfigurationChange, &configurationChange{
		Service: service,
		Setting: setting,
	})
}

type configurationChange struct {
	Service fimptype.ServiceNameT
	Setting string
}

// PublishConfigurationChanges publishes a configuration change event for every changed setting.
func PublishConfigurationChanges(eventManager event.Manager, service fimptype.ServiceNameT, settings ...string) {
	for _, setting := range settings {
		eventManager.Publish(NewConfigurationChangeEvent(service, setting))
	}
}

func WaitForConfigurationUpdate(service fimptype.ServiceNameT, setting string) event.Filter {
	return event.And(
		event.WaitForDomain(eventDomain),
		event.WaitForClass(eventClassConfigurationChange),
		event.WaitForPayload(&configurationChange{
			Service: service,
			Setting: setting,
		}),
	)
}
