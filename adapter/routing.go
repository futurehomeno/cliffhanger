package adapter

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/router"
)

const (
	CmdThingGetInclusionReport = "cmd.thing.get_inclusion_report"
	EvtThingInclusionReport    = "evt.thing.inclusion_report"
	EvtThingExclusionReport    = "evt.thing.exclusion_report"
	CmdThingDelete             = "cmd.thing.delete"
)

// RouteAdapter adds routing for adapter specific commands.
func RouteAdapter(adapter Adapter, deleteCallback func(thing Thing)) []*router.Routing {
	return []*router.Routing{
		RouteCmdThingGetInclusionReport(adapter),
		RouteCmdThingDelete(adapter, deleteCallback),
	}
}

// RouteCmdThingGetInclusionReport returns a routing responsible for handling the command.
func RouteCmdThingGetInclusionReport(adapter Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdThingGetInclusionReport(adapter),
		router.ForService(adapter.Name()),
		router.ForType(CmdThingGetInclusionReport),
	)
}

// HandleCmdThingGetInclusionReport returns a handler responsible for handling the command.
func HandleCmdThingGetInclusionReport(adapter Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			address, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: provided address has an incorrect format: %w", err)
			}

			t := adapter.ThingByAddress(address)
			if t == nil {
				return nil, fmt.Errorf("adapter: thing not found under the provided address: %s", address)
			}

			err = adapter.SendInclusionReport(t)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send the inclusion report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdThingDelete returns a routing responsible for handling the command.
func RouteCmdThingDelete(adapter Adapter, deleteCallback func(thing Thing)) *router.Routing {
	return router.NewRouting(
		HandleCmdThingDelete(adapter, deleteCallback),
		router.ForService(adapter.Name()),
		router.ForType(CmdThingDelete),
	)
}

// HandleCmdThingDelete returns a handler responsible for handling the command.
func HandleCmdThingDelete(adapter Adapter, deleteCallback func(thing Thing)) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			value, err := message.Payload.GetStrMapValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: provided address has an incorrect format: %w", err)
			}

			address := value["address"]

			t := adapter.ThingByAddress(address)
			if t == nil {
				return nil, fmt.Errorf("adapter: thing not found under the provided address: %s", address)
			}

			err = adapter.RemoveThing(address)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send the exclusion report: %w", err)
			}

			if deleteCallback != nil {
				deleteCallback(t)
			}

			return nil, nil
		}),
	)
}
