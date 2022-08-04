package scenectrl

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing service, commands and events.
const (
	CmdSceneGetReport = "cmd.scene.get_report"
	CmdSceneSet       = "cmd.scene.set"
	EvtSceneReport    = "evt.scene.report"

	SceneCtrl = "scene_ctrl"
)

// RouteService returns routing for service specific commands.
func RouteService(adapter adapter.Adapter) []*router.Routing {
	return []*router.Routing{
		RouteCmdSceneSet(adapter),
		RouteCmdSceneGetReport(adapter),
	}
}

// RouteCmdSceneSet returns a routing responsible for handling the command.
func RouteCmdSceneSet(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdSceneSet(adapter),
		router.ForService(SceneCtrl),
		router.ForType(CmdSceneSet),
	)
}

// HandleCmdSceneSet returns a handler responsible for handling the command.
func HandleCmdSceneSet(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			scene, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			sceneID, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to parse scene id: %w", err)
			}

			err = scene.SetScene(sceneID)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to set scene: %w", err)
			}

			_, err = scene.SendSceneReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send scene report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdSceneGetReport returns a routing responsible for handling the command.
func RouteCmdSceneGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdSceneGetReport(adapter),
		router.ForService(SceneCtrl),
		router.ForType(CmdSceneGetReport),
	)
}

// HandleCmdSceneGetReport returns a handler responsible for handling the command.
func HandleCmdSceneGetReport(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			scene, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := scene.SendSceneReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send presence report: %w", err)
			}

			return nil, nil
		}),
	)
}
