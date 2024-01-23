package virtualmeter

import (
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/event"
)

type (
	processor struct {
		manager *manager
	}
)

var _ event.Processor = (*processor)(nil)

// NewHandler creates a new handler for virtual meter that listens for the state updates of other services.
func NewHandler(manager *manager) *event.Handler {
	filter := event.Or(event.WaitFor[*outlvlswitch.LevelEvent](), event.WaitFor[*adapter.ConnectivityEvent]())

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
		log.Warnf("listener: received an event of type %T, expected *outlvlswitch.LevelEvent or adapter.ConnectivityEvent", e)
	}
}

func (p *processor) handleLevelEvent(levelEvent *outlvlswitch.LevelEvent) {
	mode := ModeOn
	if levelEvent.Level == 0 {
		mode = ModeOff
	}

	vmsAddr, err := p.manager.vmsAddressFromTopic(levelEvent.Address())
	if err != nil {
		log.WithError(err).Errorf("listener: failed to get virtual meter address by topic %s", levelEvent.Address())

		return
	}

	if !levelEvent.HasChanged() && !p.manager.updateRequired(vmsAddr) {
		return
	}

	level := p.manager.normalizeOutLvlSwitchLevel(levelEvent.Level, levelEvent.Address())

	if err := p.manager.update(vmsAddr, mode, level); err != nil {
		log.WithError(err).Errorf("listener: failed to update virtual meter with mode %s and level %v", mode, levelEvent.Level)
	}
}

func (p *processor) handleConnectivityEvent(connectivityEvent *adapter.ConnectivityEvent) {
	active := true
	if connectivityEvent.Connectivity.ConnectionStatus == adapter.ConnectionStatusDown {
		active = false
	}

	if err := p.manager.updateDeviceActivity(connectivityEvent.Address(), active); err != nil {
		log.WithError(err).Errorf("listener: failed to update virtual meter with active %v", active)
	}
}
