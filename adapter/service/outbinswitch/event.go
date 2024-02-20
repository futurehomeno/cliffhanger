package outbinswitch

import (
	"github.com/futurehomeno/cliffhanger/adapter"
)

// BinaryEvent represents a binary event.
type BinaryEvent struct {
	adapter.ServiceEvent

	state bool
}

func newBinaryEvent(eventType string, hasChanged bool, value bool) *BinaryEvent {
	return &BinaryEvent{
		ServiceEvent: adapter.NewServiceEvent(eventType, hasChanged),
		state:        value,
	}
}
