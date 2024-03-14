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

	levelEventProcessor struct {
		processor
	}

	connectivityEventProcessor struct {
		processor
	}
)

var (
	_ event.Processor = (*levelEventProcessor)(nil)
	_ event.Processor = (*connectivityEventProcessor)(nil)
)

// NewHandlers creates a new handler for virtual meter that listens for the state updates of other services.
func NewHandlers(mr Manager) []*event.Handler {
	m, ok := mr.(*manager)
	if !ok {
		log.Errorf("listener: failed to cast manager to *manager during handler creation")

		return nil
	}

	return []*event.Handler{
		event.NewHandler(&levelEventProcessor{processor{manager: m}}, "virtual_meter_level", 3, outlvlswitch.WaitForLevelEvent()),
		event.NewHandler(&connectivityEventProcessor{processor{manager: m}}, "virtual_meter_connectivity", 3, adapter.WaitForConnectivityEvent()),
	}
}

// Process processes events related to the level of the virtual meter.
// It updates the virtual meter if the level data is changed.
func (p *levelEventProcessor) Process(e event.Event) {
	levelEvent, ok := e.(*outlvlswitch.LevelEvent)
	if !ok {
		log.Warnf("listener: received an event of type %T, expected *outlvlswitch.LevelEvent", e)

		return
	}

	mode := ModeOn
	if levelEvent.Level == 0 {
		mode = ModeOff
	}

	vmsAddr, err := p.manager.vmsAddressFromTopic(levelEvent.Address())
	if err != nil {
		log.WithError(err).Errorf("listener: failed to get virtual meter address by topic %s", levelEvent.Address())

		return
	}

	level, err := p.manager.normalizeOutLvlSwitchLevel(levelEvent.Level, levelEvent.Address())
	if err != nil {
		log.WithError(err).Errorf("listener: failed to normalize level %v", levelEvent.Level)

		return
	}

	if err := p.manager.update(vmsAddr, mode, level); err != nil {
		log.WithError(err).Errorf("listener: failed to update virtual meter with mode %s and level %v", mode, levelEvent.Level)
	}
}

// Process processes events related to the connectivity of the virtual meter.
// It updates the virtual meter if the connectivity data is changed.
func (p *connectivityEventProcessor) Process(e event.Event) {
	connectivityEvent, ok := e.(*adapter.ConnectivityEvent)
	if !ok {
		log.Warnf("listener: received an event of type %T, expected *adapter.ConnectivityEvent", e)

		return
	}

	active := true
	if connectivityEvent.Connectivity.ConnectionStatus == adapter.ConnectionStatusDown {
		active = false
	}

	if err := p.manager.updateDeviceActivity(connectivityEvent.Address(), active); err != nil {
		log.WithError(err).Errorf("listener: failed to update virtual meter with active %v", active)
	}
}
