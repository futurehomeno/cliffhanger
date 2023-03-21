package adapter

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/router"
)

const (
	CmdThingGetInclusionReport = "cmd.thing.get_inclusion_report"
	EvtThingInclusionReport    = "evt.thing.inclusion_report"
	EvtThingExclusionReport    = "evt.thing.exclusion_report"
	CmdThingDelete             = "cmd.thing.delete"
	CmdNetworkGetNode          = "cmd.network.get_node"
	EvtNetworkNodeReport       = "evt.network.node_report"
	CmdNetworkGetAllNodes      = "cmd.network.get_all_nodes"
	EvtNetworkAllNodesReport   = "evt.network.all_nodes_report"
	CmdNetworkReset            = "cmd.network.reset"
	EvtNetworkResetDone        = "evt.network.reset_done"
	CmdPingSend                = "cmd.ping.send"
	EvtPingReport              = "evt.ping.report"
)

// RouteAdapter returns routing for adapter specific commands.
func RouteAdapter(adapter Adapter) []*router.Routing {
	return []*router.Routing{
		RouteCmdThingGetInclusionReport(adapter),
		RouteCmdThingDelete(adapter),
		RouteCmdNetworkGetNode(adapter),
		RouteCmdNetworkGetAllNodes(adapter),
		RouteCmdPingSend(adapter),
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
			t, err := getThingByMessage(adapter, message)
			if err != nil {
				return nil, err
			}

			_, err = t.SendInclusionReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send the inclusion report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdThingDelete returns a routing responsible for handling the command.
func RouteCmdThingDelete(adapter Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdThingDelete(adapter),
		router.ForService(adapter.Name()),
		router.ForType(CmdThingDelete),
	)
}

// HandleCmdThingDelete returns a handler responsible for handling the command.
func HandleCmdThingDelete(adapter Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			value, err := message.Payload.GetStrMapValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: provided address has an incorrect format: %w", err)
			}

			address := value["address"]

			t := adapter.ThingByAddress(address)
			if t == nil {
				log.Warnf("adapter: thing not found under the provided address %s, sending exclusion report regardless...", address)

				err = adapter.SendExclusionReport(address)
				if err != nil {
					return nil, fmt.Errorf("adapter: failed to send the exclusion report: %w", err)
				}

				return nil, nil
			}

			id, ok := adapter.ExchangeAddress(address)
			if ok {
				err = adapter.DestroyThing(id)
				if err != nil {
					return nil, fmt.Errorf("adapter: failed to delete thing: %w", err)
				}
			} else {
				err = adapter.RemoveThing(address)
				if err != nil {
					return nil, fmt.Errorf("adapter: failed to send the exclusion report: %w", err)
				}
			}

			return nil, nil
		}),
	)
}

// RouteCmdNetworkGetNode returns a routing responsible for handling the command.
func RouteCmdNetworkGetNode(adapter Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdNetworkGetNode(adapter),
		router.ForService(adapter.Name()),
		router.ForType(CmdNetworkGetNode),
	)
}

// HandleCmdNetworkGetNode returns a handler responsible for handling the command.
func HandleCmdNetworkGetNode(adapter Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			t, err := getThingByMessage(adapter, message)
			if err != nil {
				return nil, err
			}

			_, err = t.SendConnectivityReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send the node report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdNetworkGetAllNodes returns a routing responsible for handling the command.
func RouteCmdNetworkGetAllNodes(adapter Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdNetworkGetAllNodes(adapter),
		router.ForService(adapter.Name()),
		router.ForType(CmdNetworkGetAllNodes),
	)
}

// HandleCmdNetworkGetAllNodes returns a handler responsible for handling the command.
func HandleCmdNetworkGetAllNodes(adapter Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			err = adapter.SendAllNodesReport()
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send all nodes report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdPingSend returns a routing responsible for handling the command.
func RouteCmdPingSend(adapter Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdPingSend(adapter),
		router.ForService(adapter.Name()),
		router.ForType(CmdPingSend),
	)
}

// HandleCmdPingSend returns a handler responsible for handling the command.
func HandleCmdPingSend(adapter Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			t, err := getThingByMessage(adapter, message)
			if err != nil {
				return nil, err
			}

			err = t.SendPingReport()
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send the ping report: %w", err)
			}

			return nil, nil
		}),
	)
}

func getThingByMessage(adapter Adapter, message *fimpgo.Message) (Thing, error) {
	address, err := message.Payload.GetStringValue()
	if err != nil {
		return nil, fmt.Errorf("adapter: provided address has an incorrect format: %w", err)
	}

	t := adapter.ThingByAddress(address)
	if t == nil {
		return nil, fmt.Errorf("adapter: thing not found under the provided address: %s", address)
	}

	return t, nil
}
