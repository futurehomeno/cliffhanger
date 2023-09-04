package devsys

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing service, commands and events.
const (
	CmdThingReboot = "cmd.thing.reboot"

	DevSys = "dev_sys"
)

// RouteService returns routing for service specific commands.
func RouteService(serviceRegistry adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		routeCmdThingReboot(serviceRegistry),
	}
}

// routeCmdThingReboot returns a routing responsible for handling the command.
func routeCmdThingReboot(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdThingReboot(serviceRegistry),
		router.ForService(DevSys),
		router.ForType(CmdThingReboot),
	)
}

// handleCmdThingReboot returns a handler responsible for handling the command.
func handleCmdThingReboot(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			devSys, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			// We do not need to handle the error, as for backwards compatibility we should consider all other reboot commands as soft reboots.
			hard, _ := message.Payload.GetBoolValue()

			err = devSys.Reboot(hard)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to reboot: %w", err)
			}

			return nil, nil
		}),
		router.WithSuccessConfirmation(),
	)
}

// getService returns a service responsible for handling the message.
func getService(serviceRegistry adapter.ServiceRegistry, message *fimpgo.Message) (Service, error) {
	s := serviceRegistry.ServiceByTopic(message.Topic)
	if s == nil {
		return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
	}

	chargepoint, ok := s.(Service)
	if !ok {
		return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
	}

	return chargepoint, nil
}
