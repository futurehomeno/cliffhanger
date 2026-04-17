package diagnostic

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

const (
	CmdLQIGetReport = "cmd.lqi.get_report"
	EvtLQIReport    = "evt.lqi.report"

	CmdRSSIGetReport = "cmd.rssi.get_report"
	EvtRSSIReport    = "evt.rssi.report"

	CmdRebootReasonGetReport = "cmd.reboot_reason.get_report"
	EvtRebootReasonReport    = "evt.reboot_reason.report"

	CmdRebootsCountGetReport = "cmd.reboots_count.get_report"
	EvtRebootCountReport     = "evt.reboot_count.report"

	Diagnostic = "diagnostic"
)

// RouteService returns routing for service specific commands.
func RouteService(serviceRegistry adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		routeCmdLQIGetReport(serviceRegistry),
		routeCmdRSSIGetReport(serviceRegistry),
		routeCmdRebootReasonGetReport(serviceRegistry),
		routeCmdRebootsCountGetReport(serviceRegistry),
	}
}

// routeCmdLQIGetReport returns a routing responsible for handling the command.
func routeCmdLQIGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleSendReport(serviceRegistry, func(s Service) (bool, error) { return s.SendLQIReport(true) }),
		router.ForService(Diagnostic),
		router.ForType(CmdLQIGetReport),
	)
}

// routeCmdRSSIGetReport returns a routing responsible for handling the command.
func routeCmdRSSIGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleSendReport(serviceRegistry, func(s Service) (bool, error) { return s.SendRSSIReport(true) }),
		router.ForService(Diagnostic),
		router.ForType(CmdRSSIGetReport),
	)
}

// routeCmdRebootReasonGetReport returns a routing responsible for handling the command.
func routeCmdRebootReasonGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleSendReport(serviceRegistry, func(s Service) (bool, error) { return s.SendRebootReasonReport(true) }),
		router.ForService(Diagnostic),
		router.ForType(CmdRebootReasonGetReport),
	)
}

// routeCmdRebootsCountGetReport returns a routing responsible for handling the command.
func routeCmdRebootsCountGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleSendReport(serviceRegistry, func(s Service) (bool, error) { return s.SendRebootsCountReport(true) }),
		router.ForService(Diagnostic),
		router.ForType(CmdRebootsCountGetReport),
	)
}

// handleSendReport returns a handler that sends a diagnostic report using the provided send function.
func handleSendReport(serviceRegistry adapter.ServiceRegistry, send func(Service) (bool, error)) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			if _, err := send(s); err != nil {
				return nil, fmt.Errorf("failed to send diagnostic report: %w", err)
			}

			return nil, nil
		}),
	)
}

// getService returns a service responsible for handling the message.
func getService(serviceRegistry adapter.ServiceRegistry, message *fimpgo.Message) (Service, error) {
	s := serviceRegistry.ServiceByTopic(message.Topic)
	if s == nil {
		return nil, fmt.Errorf("service not found under the provided address: %s", message.Addr.ServiceAddress)
	}

	diagnostic, ok := s.(Service)
	if !ok {
		return nil, fmt.Errorf("incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
	}

	return diagnostic, nil
}
