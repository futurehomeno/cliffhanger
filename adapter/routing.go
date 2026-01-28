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
	CmdNetworkReset            = "cmd.network.reset"
	EvtNetworkResetDone        = "evt.network.reset_done"
	CmdNetworkGetNode          = "cmd.network.get_node"
	EvtNetworkNodeReport       = "evt.network.node_report"
	CmdNetworkGetAllNodes      = "cmd.network.get_all_nodes"
	EvtNetworkAllNodesReport   = "evt.network.all_nodes_report"
	CmdPingSend                = "cmd.ping.send"
	EvtPingReport              = "evt.ping.report"
)

// RouteAdapter returns routing for adapter specific commands.
func RouteAdapter(adapter Adapter) []*router.Routing {
	return []*router.Routing{
		routeCmdThingGetInclusionReport(adapter),
		routeCmdThingDelete(adapter),
		routeCmdNetworkReset(adapter),
		routeCmdNetworkGetNode(adapter),
		routeCmdNetworkGetAllNodes(adapter),
		routeCmdPingSend(adapter),
	}
}

// routeCmdThingGetInclusionReport returns a routing responsible for handling the command.
func routeCmdThingGetInclusionReport(adapter Adapter) *router.Routing {
	return router.NewRouting(
		handleCmdThingGetInclusionReport(adapter),
		router.ForService(adapter.Name()),
		router.ForType(CmdThingGetInclusionReport),
	)
}

// handleCmdThingGetInclusionReport returns a handler responsible for handling the command.
func handleCmdThingGetInclusionReport(adapter Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			t, err := getThingByMessage(adapter, message)
			if err != nil {
				return nil, err
			}

			_, err = t.SendInclusionReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send the inclusion report: %w", err)
			}

			return nil, nil
		}),
	)
}

// routeCmdThingDelete returns a routing responsible for handling the command.
func routeCmdThingDelete(adapter Adapter) *router.Routing {
	return router.NewRouting(
		handleCmdThingDelete(adapter),
		router.ForService(adapter.Name()),
		router.ForType(CmdThingDelete),
	)
}

// handleCmdThingDelete returns a handler responsible for handling the command.
func handleCmdThingDelete(adapter Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			value, err := message.Payload.GetStrMapValue()
			if err != nil {
				return nil, fmt.Errorf("provided address has an incorrect format: %w", err)
			}

			address := value["address"]

			err = adapter.DestroyThingByAddress(address)
			if err != nil {
				return nil, fmt.Errorf("failed to delete thing with address %s: %w", address, err)
			}

			return nil, nil
		}),
	)
}

// routeCmdNetworkReset returns a routing responsible for handling the command.
func routeCmdNetworkReset(adapter Adapter) *router.Routing {
	return router.NewRouting(
		handleCmdNetworkReset(adapter),
		router.ForService(adapter.Name()),
		router.ForType(CmdNetworkReset),
	)
}

// handleCmdNetworkReset returns a handler responsible for handling the command.
func handleCmdNetworkReset(adapter Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			err = adapter.DestroyAllThings()
			if err != nil {
				return nil, fmt.Errorf("failed to reset all things: %w", err)
			}

			return fimpgo.NewNullMessage(
				EvtNetworkResetDone,
				adapter.Name(),
				nil,
				nil,
				message.Payload,
			), nil
		}),
	)
}

// routeCmdNetworkGetNode returns a routing responsible for handling the command.
func routeCmdNetworkGetNode(adapter Adapter) *router.Routing {
	return router.NewRouting(
		handleCmdNetworkGetNode(adapter),
		router.ForService(adapter.Name()),
		router.ForType(CmdNetworkGetNode),
	)
}

// handleCmdNetworkGetNode returns a handler responsible for handling the command.
func handleCmdNetworkGetNode(adapter Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			t, err := getThingByMessage(adapter, message)
			if err != nil {
				return nil, err
			}

			_, err = t.SendConnectivityReport(true)
			if err != nil {
				return nil, fmt.Errorf("failed to send the node report: %w", err)
			}

			return nil, nil
		}),
	)
}

// routeCmdNetworkGetAllNodes returns a routing responsible for handling the command.
func routeCmdNetworkGetAllNodes(adapter Adapter) *router.Routing {
	return router.NewRouting(
		handleCmdNetworkGetAllNodes(adapter),
		router.ForService(adapter.Name()),
		router.ForType(CmdNetworkGetAllNodes),
	)
}

// handleCmdNetworkGetAllNodes returns a handler responsible for handling the command.
func handleCmdNetworkGetAllNodes(adapter Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(_ *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			err = adapter.SendConnectivityReport()
			if err != nil {
				return nil, fmt.Errorf("failed to send connectivity report: %w", err)
			}

			return nil, nil
		}),
	)
}

// routeCmdPingSend returns a routing responsible for handling the command.
func routeCmdPingSend(adapter Adapter) *router.Routing {
	return router.NewRouting(
		handleCmdPingSend(adapter),
		router.ForService(adapter.Name()),
		router.ForType(CmdPingSend),
	)
}

// handleCmdPingSend returns a handler responsible for handling the command.
func handleCmdPingSend(adapter Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			t, err := getThingByMessage(adapter, message)
			if err != nil {
				return nil, err
			}

			err = t.SendPingReport()
			if err != nil {
				return nil, fmt.Errorf("failed to send the ping report: %w", err)
			}

			return nil, nil
		}),
	)
}

func getThingByMessage(adapter Adapter, message *fimpgo.Message) (Thing, error) {
	address, err := message.Payload.GetStringValue()
	if err != nil {
		return nil, fmt.Errorf("provided address has an incorrect format: %w", err)
	}

	t := adapter.ThingByAddress(address)
	if t == nil {
		return nil, fmt.Errorf("thing not found under the provided address: %s", address)
	}

	return t, nil
}
