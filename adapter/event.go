package adapter

import (
	"fmt"

	"github.com/futurehomeno/cliffhanger/event"
)

func ServiceEventDomain(serviceName string) string {
	return fmt.Sprintf("adapter_service_%s", serviceName)
}

func GetServiceEvent(e *event.Event) *ServiceEvent {
	serviceEvent, _ := e.Payload.(*ServiceEvent)

	return serviceEvent
}

type ServiceEvent struct {
	Address    string
	Event      string
	HasChanged bool
	Payload    interface{}
}
