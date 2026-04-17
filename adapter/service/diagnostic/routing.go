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
	EvtRebootsCountReport    = "evt.reboots_count.report"

	CmdUptimeGetReport = "cmd.uptime.get_report"
	EvtUptimeReport    = "evt.uptime.report"

	CmdErrorsGetReport = "cmd.errors.get_report"
	EvtErrorsReport    = "evt.errors.report"

	Diagnostic = "diagnostic"
)

func RouteService(serviceRegistry adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		routeCmdLQIGetReport(serviceRegistry),
		routeCmdRSSIGetReport(serviceRegistry),
		routeCmdRebootReasonGetReport(serviceRegistry),
		routeCmdRebootsCountGetReport(serviceRegistry),
		routeCmdUptimeGetReport(serviceRegistry),
		routeCmdErrorsGetReport(serviceRegistry),
	}
}

func routeCmdLQIGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleSendReport(serviceRegistry, Service.SendLQIReport),
		router.ForService(Diagnostic),
		router.ForType(CmdLQIGetReport),
	)
}

func routeCmdRSSIGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleSendReport(serviceRegistry, Service.SendRSSIReport),
		router.ForService(Diagnostic),
		router.ForType(CmdRSSIGetReport),
	)
}

func routeCmdRebootReasonGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleSendReport(serviceRegistry, Service.SendRebootReasonReport),
		router.ForService(Diagnostic),
		router.ForType(CmdRebootReasonGetReport),
	)
}

func routeCmdRebootsCountGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleSendReport(serviceRegistry, Service.SendRebootsCountReport),
		router.ForService(Diagnostic),
		router.ForType(CmdRebootsCountGetReport),
	)
}

func routeCmdUptimeGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleSendReport(serviceRegistry, Service.SendUptimeReport),
		router.ForService(Diagnostic),
		router.ForType(CmdUptimeGetReport),
	)
}

func routeCmdErrorsGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleSendReport(serviceRegistry, Service.SendErrorsReport),
		router.ForService(Diagnostic),
		router.ForType(CmdErrorsGetReport),
	)
}

func handleSendReport(serviceRegistry adapter.ServiceRegistry, send func(Service) error) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s, err := getService(serviceRegistry, message)
			if err != nil {
				return nil, err
			}

			if err := send(s); err != nil {
				return nil, fmt.Errorf("failed to send diagnostic report: %w", err)
			}

			return nil, nil
		}),
	)
}

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
