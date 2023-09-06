package discovery

import (
	"context"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining service discovery routing.
const (
	Topic   = "pt:j1/mt:cmd/rt:discovery"
	Service = "system"

	CmdDiscoveryRequest = "cmd.discovery.request"
	EvtDiscoveryReport  = "evt.discovery.report"
)

// Route returns a routing responsible for handling the command.
func Route(resource *Resource) *router.Routing {
	return router.NewRouting(
		Handle(resource),
		router.ForTopic(Topic),
		router.ForType(CmdDiscoveryRequest),
	)
}

// Handle returns a handler responsible for handling the command.
func Handle(resource *Resource) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(ctx context.Context, message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			return fimpgo.NewObjectMessage(
				EvtDiscoveryReport,
				Service,
				resource,
				nil,
				nil,
				message.Payload,
			), nil
		}),
	)
}
