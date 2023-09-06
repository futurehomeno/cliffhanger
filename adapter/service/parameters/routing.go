package parameters

import (
	"context"
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing service, commands and events.
const (
	CmdSupParamsGetReport = "cmd.sup_params.get_report"
	EvtSupParamsReport    = "evt.sup_params.report"
	CmdParamSet           = "cmd.param.set"
	CmdParamGetReport     = "cmd.param.get_report"
	EvtParamReport        = "evt.param.report"

	Parameters = "parameters"
)

// RouteService returns routing for service specific commands.
func RouteService(serviceRegistry adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		routeCmdSupParamsGetReport(serviceRegistry),
		routeCmdParamSet(serviceRegistry),
		routeCmdParamGetReport(serviceRegistry),
	}
}

// routeCmdSupParamsGetReport returns a routing responsible for handling the command.
func routeCmdSupParamsGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdSupParamsGetReport(serviceRegistry),
		router.ForService(Parameters),
		router.ForType(CmdSupParamsGetReport),
	)
}

// handleCmdSupParamsGetReport returns a handler responsible for handling the command.
func handleCmdSupParamsGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(ctx context.Context, message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			parameters, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			_, err = parameters.SendSupportedParamsReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send supported parameters report: %w", err)
			}

			return nil, nil
		}),
	)
}

// routeCmdParamSet returns a routing responsible for handling the command.
func routeCmdParamSet(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdParamSet(serviceRegistry),
		router.ForService(Parameters),
		router.ForType(CmdParamSet),
	)
}

// handleCmdParamSet returns a handler responsible for handling the command.
func handleCmdParamSet(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(ctx context.Context, message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			parameters, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			var param Parameter

			if err := message.Payload.GetObjectValue(&param); err != nil {
				return nil, fmt.Errorf("adapter: provided parameter has an incorrect format: %w", err)
			}

			if err := parameters.SetParameter(&param); err != nil {
				return nil, fmt.Errorf("adapter: failed to set a parameter: %w", err)
			}

			if _, err := parameters.SendParameterReport(param.ID, true); err != nil {
				return nil, fmt.Errorf("adapter: failed to send parameter report: %w", err)
			}

			return nil, nil
		}),
	)
}

// routeCmdParamGetReport returns a routing responsible for handling the command.
func routeCmdParamGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdParamGetReport(serviceRegistry),
		router.ForService(Parameters),
		router.ForType(CmdParamGetReport),
	)
}

// handleCmdParamGetReport returns a handler responsible for handling the command.
func handleCmdParamGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(ctx context.Context, message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			parameters, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			value, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: provided parameter id has an incorrect format: %w", err)
			}

			if _, err = parameters.SendParameterReport(value, true); err != nil {
				return nil, fmt.Errorf("adapter: failed to send parameter report: %w", err)
			}

			return nil, nil
		}),
	)
}

// getService returns a service responsible for handling the message.
func getService(serviceRegistry adapter.ServiceRegistry, message *fimpgo.Message) (Service, error) {
	s := serviceRegistry.ServiceByTopic(message.Topic)
	if s == nil {
		return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
	}

	parameters, ok := s.(Service)
	if !ok {
		return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
	}

	return parameters, nil
}
