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
func NewHandlers(mr Manager, handlersBufferSize int) []*event.Handler {
	m, ok := mr.(*manager)
	if !ok {
		log.Errorf("[cliff] Manager cast failed")

		return nil
	}

	return []*event.Handler{
		event.NewHandler(&levelEventProcessor{processor{manager: m}}, "virtual_meter_level", handlersBufferSize, outlvlswitch.WaitForLevelEvent(), adapter.WaitForChange()),
		event.NewHandler(&connectivityEventProcessor{processor{manager: m}}, "virtual_meter_connectivity", handlersBufferSize, adapter.WaitForConnectivityEvent()),
	}
}

// Process processes events related to the level of the virtual meter.
// It updates the virtual meter if the level data is changed.
func (p *levelEventProcessor) Process(e event.Event) {
	levelEvent, ok := e.(*outlvlswitch.LevelEvent)
	if !ok {
		log.Warnf("[cliff] Received event type=%T exp=*outlvlswitch.LevelEvent", e)
		return
	}

	mode := ModeOn
	if levelEvent.Level == 0 {
		mode = ModeOff
	}

	vmsAddr, err := p.manager.vmsAddressFromTopic(levelEvent.Address())
	if err != nil {
		log.Errorf("[cliff] Find vms address. topic: %s err: %v", levelEvent.Address(), err)

		return
	}

	level, err := p.manager.normalizeOutLvlSwitchLevel(levelEvent.Level, levelEvent.Address())
	if err != nil {
		log.Errorf("[cliff] Normalize level. level: %v err: %v", levelEvent.Level, err)

		return
	}

	if err := p.manager.update(vmsAddr, mode, level); err != nil {
		log.Errorf("[cliff] Update vm. mode: %s level: %v err: %v", mode, levelEvent.Level, err)
	}
}

// Process processes events related to the connectivity of the virtual meter.
// It updates the virtual meter if the connectivity data is changed.
func (p *connectivityEventProcessor) Process(e event.Event) {
	connectivityEvent, ok := e.(*adapter.ConnectivityEvent)
	if !ok {
		log.Warnf("[cliff] Received event type=%T exp=*adapter.ConnectivityEvent", e)
		return
	}

	active := connectivityEvent.Connectivity.ConnStatus != adapter.ConnStatusDown

	if err := p.manager.updateDeviceActivity(connectivityEvent.Address(), active); err != nil {
		log.Errorf("[cliff] Update vm activity. active: %v err: %v", active, err)
	}
}
