package discovery

import (
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/lifecycle"
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
// appLifecycle may be nil; when provided, each reply includes fresh app states.
func Route(resourceName fimptype.ResourceNameT, resourceType fimptype.ResourceTypeT, packageName, instanceID, version string, appLifecycle *lifecycle.Lifecycle) *router.Routing {
	return router.NewRouting(
		Handle(resourceName, resourceType, packageName, instanceID, version, appLifecycle),
		router.ForTopic(Topic),
		router.ForType(CmdDiscoveryRequest),
	)
}

// Handle returns a handler responsible for handling the command.
// appLifecycle may be nil; when provided, each reply includes fresh app states.
func Handle(resourceName fimptype.ResourceNameT, resourceType fimptype.ResourceTypeT, packageName, instanceID, version string, appLifecycle *lifecycle.Lifecycle) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			reply := &resourceT{
				ResourceName: resourceName,
				ResourceType: resourceType,
				PackageName:  packageName,
				InstanceID:   instanceID,
				Version:      version,
			}
			if appLifecycle != nil {
				reply.States = appLifecycle.AllStates()
			}

			return fimpgo.NewObjectMessage(
				EvtDiscoveryReport,
				Service,
				reply,
				nil,
				nil,
				message.Payload,
			), nil
		}),
	)
}
