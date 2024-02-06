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

func WaitForLevelEvent() event.Filter {
	return event.WaitFor[*LevelEvent]()
}
