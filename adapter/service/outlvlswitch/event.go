package outlvlswitch

import (
	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/event"
)

type LevelEvent struct {
	adapter.ServiceEvent

	Level float64
}

func newLevelEvent(eventType string, hasChanged bool, level float64) *LevelEvent {
	return &LevelEvent{
		ServiceEvent: adapter.NewServiceEvent(eventType, hasChanged),
		Level:        level,
	}
}

// WaitForLevelReport returns a filter that waits for a level report.
func WaitForLevelReport() event.Filter {
	return event.FilterFn(func(e event.Event) bool {
		_, ok := e.(*LevelEvent)

		return ok
	})
}
