package presence

import (
	"context"
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

const (
	CmdPresenceGetReport = "cmd.presence.get_report"
	EvtPresenceReport    = "evt.presence.report"

	SensorPresence = "sensor_presence"
)

// RouteService returns routing for service specific commands.
func RouteService(serviceRegistry adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		RouteCmdPresenceGetReport(serviceRegistry),
	}
}

// RouteCmdPresenceGetReport returns a routing responsible for handling the command.
func RouteCmdPresenceGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdPresenceGetReport(serviceRegistry),
		router.ForService(SensorPresence),
		router.ForType(CmdPresenceGetReport),
	)
}

// HandleCmdPresenceGetReport returns a handler responsible for handling the command.
func HandleCmdPresenceGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(ctx context.Context, message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			presence, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := presence.SendPresenceReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send presence report: %w", err)
			}

			return nil, nil
		}),
	)
}
