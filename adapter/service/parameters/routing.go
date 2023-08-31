package parameters

import (
	"errors"
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
		RouteCmdSupParamsGetReport(serviceRegistry),
		RouteCmdParamSet(serviceRegistry),
		RouteCmdParamGetReport(serviceRegistry),
	}
}

// RouteCmdSupParamsGetReport returns a routing responsible for handling the command.
func RouteCmdSupParamsGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdSupParamsGetReport(serviceRegistry),
		router.ForService(Parameters),
		router.ForType(CmdSupParamsGetReport),
	)
}

// HandleCmdSupParamsGetReport returns a handler responsible for handling the command.
func HandleCmdSupParamsGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			parameters, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			if !parameters.SupportsParamsDiscovery() {
				return nil, errors.New("adapter: service does not support parameters discovery")
			}

			err := parameters.SendSupportedParamsReport()
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send supported parameters report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdParamSet returns a routing responsible for handling the command.
func RouteCmdParamSet(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdParamSet(serviceRegistry),
		router.ForService(Parameters),
		router.ForType(CmdParamSet),
	)
}

// HandleCmdParamSet returns a handler responsible for handling the command.
func HandleCmdParamSet(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			parameters, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			var param Parameter

			if err := message.Payload.GetObjectValue(&param); err != nil {
				return nil, fmt.Errorf("adapter: provided cable lock value has an incorrect format: %w", err)
			}

			if err := parameters.SetParameter(param); err != nil {
				return nil, fmt.Errorf("adapter: failed to set a parameter: %w", err)
			}

			if err := parameters.SendParameterReport(param.ID); err != nil {
				return nil, fmt.Errorf("adapter: failed to send parameter report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdParamGetReport returns a routing responsible for handling the command.
func RouteCmdParamGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdParamGetReport(serviceRegistry),
		router.ForService(Parameters),
		router.ForType(CmdParamGetReport),
	)
}

// HandleCmdParamGetReport returns a handler responsible for handling the command.
func HandleCmdParamGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			parameters, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			value, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: provided parameter id has an incorrect format: %w", err)
			}

			err = parameters.SendParameterReport(value)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send parameter report: %w", err)
			}

			return nil, nil
		}),
	)
}
