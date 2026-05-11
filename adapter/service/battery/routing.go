package battery

import (
	"fmt"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/fimpgo"
)

const (
	CmdLevelGetReport = "cmd.lvl.get_report"
	EvtLevelReport    = "evt.lvl.report"
	EvtAlarmReport    = "evt.alarm.report"

	Battery = "battery"
)

func RouteService(serviceRegistry adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		routeCmdLevelGetReport(serviceRegistry),
	}
}

func routeCmdLevelGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdLevelGetReport(serviceRegistry),
		router.ForService(Battery),
		router.ForType(CmdLevelGetReport),
	)
}

func handleCmdLevelGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			battery, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := battery.SendBatteryLevelReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send battery level report: %w", err)
			}

			return nil, nil
		}),
	)
}
