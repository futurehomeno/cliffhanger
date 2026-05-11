package ota

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

const (
	CmdOTAUpdateStart    = "cmd.ota_update.start"
	EvtOTAStartReport    = "evt.ota_start.report"
	EvtOTAProgressReport = "evt.ota_progress.report"
	EvtOTAEndReport      = "evt.ota_end.report"

	OTA = "ota"
)

func RouteService(serviceRegistry adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		routeCmdOTAUpdateStart(serviceRegistry),
	}
}

func routeCmdOTAUpdateStart(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdOTAUpdateStart(serviceRegistry),
		router.ForService(OTA),
		router.ForType(CmdOTAUpdateStart),
	)
}

func handleCmdOTAUpdateStart(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			ota, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			firmwarePath, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("failed to get firmware path from payload: %w", err)
			}

			if err = ota.StartUpdate(firmwarePath); err != nil {
				return nil, fmt.Errorf("failed to start OTA update: %w", err)
			}

			return nil, nil
		}),
	)
}
