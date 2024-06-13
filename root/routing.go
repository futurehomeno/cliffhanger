package root

import (
	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing events and topics.
const (
	GatewayEvtTopic = "pt:j1/mt:evt/rt:ad/rn:gateway/ad:1"

	EvtGatewayFactoryReset = "evt.gateway.factory_reset"
)

// routeFactoryReset prepares routing for factory reset event.
func routeFactoryReset(rootApp App) *router.Routing {
	return router.NewRouting(
		handleFactoryReset(rootApp),
		router.ForTopic(GatewayEvtTopic),
		router.ForType(EvtGatewayFactoryReset),
	)
}

// handleFactoryReset handles factory reset event.
func handleFactoryReset(rootApp App) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(_ *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			// Factory reset requires stopping the application first, which includes stopping the message router.
			// In order to avoid a deadlock we need to run the reset in a separate goroutine, so the message router can be stopped.
			go func() {
				_ = rootApp.Reset()
			}()

			return nil, nil
		}), router.WithSilentErrors(),
	)
}
