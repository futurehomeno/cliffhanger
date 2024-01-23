package outlvlswitch

import (
	"github.com/futurehomeno/cliffhanger/adapter"
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
