package observer

import (
	"context"
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/prime"
	"github.com/futurehomeno/cliffhanger/router"
)

// RouteObserver returns routing for prime observer.
func RouteObserver(observer Observer) []*router.Routing {
	return []*router.Routing{
		RouteEvtPD7Notify(observer),
	}
}

// RouteEvtPD7Notify returns a routing responsible for handling the event.
func RouteEvtPD7Notify(observer Observer) *router.Routing {
	return router.NewRouting(
		HandleEvtPD7Notify(observer),
		router.ForTopic(prime.NotifyTopic),
		router.ForService(prime.ServiceName),
		router.ForType(prime.EvtPD7Notify),
	)
}

// HandleEvtPD7Notify returns a handler responsible for handling the event.
func HandleEvtPD7Notify(observer Observer) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(ctx context.Context, message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			notify, err := prime.NotifyFromMessage(message)
			if err != nil {
				return nil, fmt.Errorf("observer: failed to read a prime notification: %w", err)
			}

			err = observer.Update(notify)
			if err != nil {
				return nil, fmt.Errorf("observer: failed to process a prime notification: %w", err)
			}

			return nil, nil
		}),
		router.WithSilentErrors(),
	)
}
