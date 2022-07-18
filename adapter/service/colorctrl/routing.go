package colorctrl

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing service, commands and events.
const (
	CmdColorSet       = "cmd.color.set"
	CmdColorGetReport = "cmd.color.get_report"
	EvtColorReport    = "evt.color.report"

	ColorCtrl = "color_ctrl"
)

// RouteService returns routing for service specific commands.
func RouteService(adapter adapter.Adapter) []*router.Routing {
	return []*router.Routing{
		RouteCmdColorSet(adapter),
		RouteCmdColorGetReport(adapter),
	}
}

// RouteCmdColorSet returns a routing responsible for handling the command.
func RouteCmdColorSet(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdColorSet(adapter),
		router.ForService(ColorCtrl),
		router.ForType(CmdColorSet),
	)
}

// HandleCmdColorSet returns a handler responsible for handling the command.
func HandleCmdColorSet(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			colorctrl, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			color, err := message.Payload.GetIntMapValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to parse color: %w", err)
			}

			err = colorctrl.SetColor(color)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to set color: %w", err)
			}

			_, err = colorctrl.SendColorReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send color report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdColorGetReport returns a routing responsible for handling the command.
func RouteCmdColorGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdColorGetReport(adapter),
		router.ForService(ColorCtrl),
		router.ForType(CmdColorGetReport),
	)
}

// HandleCmdColorGetReport returns a handler responsible for handling the command.
func HandleCmdColorGetReport(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			colorctrl, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := colorctrl.SendColorReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send color report: %w", err)
			}

			return nil, nil
		}),
	)
}
