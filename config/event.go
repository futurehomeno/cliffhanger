package config

import (
	"github.com/futurehomeno/cliffhanger/event"
	"github.com/futurehomeno/fimpgo/fimptype"
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
