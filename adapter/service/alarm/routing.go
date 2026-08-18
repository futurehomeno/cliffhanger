package alarm

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

const (
	CmdAlarmGetReport = "cmd.alarm.get_report"
	EvtAlarmReport    = "evt.alarm.report"

	AlarmSystem = "alarm_system"
)

func RouteService(serviceRegistry adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		routeCmdAlarmGetReport(serviceRegistry),
	}
}

func routeCmdAlarmGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdAlarmGetReport(serviceRegistry),
		router.ForService(AlarmSystem),
		router.ForType(CmdAlarmGetReport),
	)
}

func handleCmdAlarmGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			alarm, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			for _, event := range alarm.SupportedEvents() {
				if _, err := alarm.SendAlarmReport(event, true); err != nil {
					return nil, fmt.Errorf("failed to send alarm report for event %s: %w", event, err)
				}
			}

			return nil, nil
		}),
	)
}
