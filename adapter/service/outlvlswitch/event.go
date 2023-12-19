package outlvlswitch

import (
	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/event"
)

func WaitForLevelReport(serviceAddress string, onChangeOnly bool) event.Filter {
	return event.FilterFn(func(e *event.Event) bool {
		if e.Domain == adapter.ServiceEventDomain(OutLvlSwitch) {
			return false
		}

		serviceEvent, ok := e.Payload.(*adapter.ServiceEvent)
		if !ok {
			return false
		}

		if serviceEvent.Address != serviceAddress {
			return false
		}

		if serviceEvent.Event != EvtLvlReport {
			return false
		}

		if onChangeOnly && !serviceEvent.HasChanged {
			return false
		}

		return true
	})
}

func GetLevel(serviceEvent *adapter.ServiceEvent) (int, bool) {
	level, ok := serviceEvent.Payload.(int)
	if !ok {
		return 0, false
	}

	return level, true
}
