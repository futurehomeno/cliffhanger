package virtualmeter

import (
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/event"
)

type (
	processor struct {
		manager Manager
	}
)

var _ event.Processor = (*processor)(nil)

// NewHandler creates a new handler for virtual meter that listens for the state updates of other services.
func NewHandler(manager Manager) *event.Handler {
	filter := event.Or(outlvlswitch.WaitForLevelReport(), adapter.WaitForThingEvent())

	return event.NewHandler(&processor{manager: manager}, "virtual_meter_level", 3, filter)

}

// Process processes events related to the virtual meter, that is:
// - outlvlswitch.LevelEvent
// - adapter.ConnectivityEvent
// Logs a warning if the event is of a different type.
func (p *processor) Process(e event.Event) {
	switch v := e.(type) {
	case *outlvlswitch.LevelEvent:
		p.handleLevelEvent(v)
	case *adapter.ConnectivityEvent:
		p.handleConnectivityEvent(v)
	default:
		log.Warnf("Received an event of type %T, expected *outlvlswitch.LevelEvent or adapter.ConnectivityEvent", e)
	}
}

func (p *processor) handleLevelEvent(levelEvent *outlvlswitch.LevelEvent) {
	mode := ModeOn
	if levelEvent.Level == 0 {
		mode = ModeOff
	}

	if !levelEvent.HasChanged() && !p.manager.UpdateRequired(levelEvent.Address()) {
		return
	}

	if err := p.manager.Update(levelEvent.Address(), mode, levelEvent.Level); err != nil {
		log.WithError(err).Errorf("Failed to update virtual meter with mode %s and level %v", mode, levelEvent.Level)
	}
}

func (p *processor) handleConnectivityEvent(connectivityEvent *adapter.ConnectivityEvent) {
	active := true
	if connectivityEvent.Connectivity.ConnectionStatus == adapter.ConnectionStatusDown {
		active = false
	}

	if err := p.manager.updateDeviceActivity(connectivityEvent.Address(), active); err != nil {
		log.WithError(err).Errorf("Failed to update virtual meter with active %v", active)
	}
}
