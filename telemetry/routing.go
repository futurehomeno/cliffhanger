package telemetry

import (
	"fmt"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/router"
)

const (
	SettingEnabled    = "telemetry_enabled"
	SettingValidity   = "telemetry_validity"
	SettingSuppressed = "telemetry_suppressed"
)

func Route(serviceName fimptype.ServiceNameT, tel Telemetry, options ...config.RoutingOption) []*router.Routing {
	return []*router.Routing{
		RouteCmdTelemetrySetEnabled(serviceName, SettingEnabled, tel, options...),
		RouteCmdTelemetryEnabled(serviceName, SettingEnabled, tel, options...),
		RouteCmdTelemetrySetValidity(serviceName, SettingValidity, tel, options...),
		RouteCmdTelemetryValidity(serviceName, SettingValidity, tel, options...),
		RouteCmdTelemetrySetSuppressedDomains(serviceName, SettingSuppressed, tel, options...),
		RouteCmdTelemetrySuppressedDomains(serviceName, SettingSuppressed, tel, options...),
	}
}

func RouteCmdTelemetryEnabled(serviceName fimptype.ServiceNameT, setting string, tel Telemetry, _ ...config.RoutingOption) *router.Routing {
	return router.NewRouting(
		router.NewMessageHandler(
			router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				return fimpgo.NewBoolMessage(
					fmt.Sprintf("evt.config.%s_report", setting),
					serviceName,
					tel.IsEnabled(),
					nil,
					nil,
					message.Payload,
				), nil
			})),
		router.ForService(serviceName),
		router.ForType("cmd.config.get_"+setting),
	)
}

func RouteCmdTelemetrySetEnabled(serviceName fimptype.ServiceNameT, setting string, tel Telemetry, _ ...config.RoutingOption) *router.Routing {
	return router.NewRouting(
		router.NewMessageHandler(
			router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				enabled, err := message.Payload.GetBoolValue()
				if err != nil {
					return nil, err
				}

				if err := tel.Enable(enabled); err != nil {
					return nil, err
				}

				return fimpgo.NewBoolMessage(
					fmt.Sprintf("evt.config.%s_report", setting),
					serviceName,
					enabled,
					nil,
					nil,
					message.Payload,
				), nil
			})),
		router.ForService(serviceName),
		router.ForType("cmd.config.set_"+setting),
	)
}

func RouteCmdTelemetryValidity(serviceName fimptype.ServiceNameT, setting string, tel Telemetry, _ ...config.RoutingOption) *router.Routing {
	return router.NewRouting(
		router.NewMessageHandler(
			router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				return fimpgo.NewStringMessage(
					fmt.Sprintf("evt.config.%s_report", setting),
					serviceName,
					tel.Validity().String(),
					nil,
					nil,
					message.Payload,
				), nil
			})),
		router.ForService(serviceName),
		router.ForType("cmd.config.get_"+setting),
	)
}

func RouteCmdTelemetrySetValidity(serviceName fimptype.ServiceNameT, setting string, tel Telemetry, _ ...config.RoutingOption) *router.Routing {
	return router.NewRouting(
		router.NewMessageHandler(
			router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				raw, err := message.Payload.GetStringValue()
				if err != nil {
					return nil, err
				}

				d, err := time.ParseDuration(raw)
				if err != nil {
					return nil, fmt.Errorf("telemetry: failed to parse validity %q: %w", raw, err)
				}

				if err := tel.SetValidity(d); err != nil {
					return nil, err
				}

				return fimpgo.NewStringMessage(
					fmt.Sprintf("evt.config.%s_report", setting),
					serviceName,
					d.String(),
					nil,
					nil,
					message.Payload,
				), nil
			})),
		router.ForService(serviceName),
		router.ForType("cmd.config.set_"+setting),
	)
}

func RouteCmdTelemetrySuppressedDomains(serviceName fimptype.ServiceNameT, setting string, tel Telemetry, _ ...config.RoutingOption) *router.Routing {
	return router.NewRouting(
		router.NewMessageHandler(
			router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				return fimpgo.NewStrArrayMessage(
					fmt.Sprintf("evt.config.%s_report", setting),
					serviceName,
					tel.SuppressedDomains(),
					nil,
					nil,
					message.Payload,
				), nil
			})),
		router.ForService(serviceName),
		router.ForType("cmd.config.get_"+setting),
	)
}

func RouteCmdTelemetrySetSuppressedDomains(serviceName fimptype.ServiceNameT, setting string, tel Telemetry, _ ...config.RoutingOption) *router.Routing {
	return router.NewRouting(
		router.NewMessageHandler(
			router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				domains, err := message.Payload.GetStrArrayValue()
				if err != nil {
					return nil, err
				}

				if err := tel.SetSuppressedDomains(domains); err != nil {
					return nil, err
				}

				return fimpgo.NewStrArrayMessage(
					fmt.Sprintf("evt.config.%s_report", setting),
					serviceName,
					domains,
					nil,
					nil,
					message.Payload,
				), nil
			})),
		router.ForService(serviceName),
		router.ForType("cmd.config.set_"+setting),
	)
}
