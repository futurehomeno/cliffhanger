package battery

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing service, commands and events.
const (
	CmdLevelGetReport = "cmd.lvl.get_report"
	EvtLevelReport    = "evt.lvl.report"
	EvtAlarmReport    = "evt.alarm.report"

	Battery = "battery"
)

// RouteService returns routing for service specific commands.
func RouteService(serviceRegistry adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		routeCmdLevelGetReport(serviceRegistry),
	}
}

// routeCmdLevelGetReport returns a routing responsible for handling the command.
func routeCmdLevelGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdLevelGetReport(serviceRegistry),
		router.ForService(Battery),
		router.ForType(CmdLevelGetReport),
	)
}

// handleCmdLevelGetReport returns a handler responsible for handling the command.
func handleCmdLevelGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			battery, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := battery.SendBatteryLevelReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send battery level report: %w", err)
			}

			return nil, nil
		}),
	)
}
