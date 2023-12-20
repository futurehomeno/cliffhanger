package outlvlswitch

import (
	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/event"
)

type LevelEvent struct {
	adapter.ServiceEvent

	Level int64
}

func newLevelEvent(eventType string, hasChanged bool, level int64) *LevelEvent {
	return &LevelEvent{
		ServiceEvent: adapter.NewServiceEvent(eventType, hasChanged),
		Level:        level,
	}
}

func WaitForLevelReport() event.Filter {
	return event.FilterFn(func(e event.Event) bool {
		_, ok := e.(*LevelEvent)
		if !ok {
			return false
		}

		return true
	})
}
