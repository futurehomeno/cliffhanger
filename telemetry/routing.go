package telemetry

import (
	"fmt"
	"time"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/telemetry/types"
)

const (
	SettingEnabled   = "telemetry_enabled"
	SettingValidity  = "telemetry_validity"
	SettingSuppressed = "telemetry_suppressed"
)

func Route(tel Telemetry, _ ...config.RoutingOption) []*router.Routing {
	if tel == nil {
		return []*router.Routing{}
	}

	return []*router.Routing{
		RouteCmdTelemetrySetEnabled(tel),
		RouteCmdTelemetryEnabled(tel),
		RouteCmdTelemetrySetValidity(tel),
		RouteCmdTelemetryValidity(tel),
		RouteCmdTelemetrySetSuppressed(tel),
		RouteCmdTelemetrySuppressed(tel),
	}
}

func RouteCmdTelemetryEnabled(tel Telemetry) *router.Routing {
	return router.NewRouting(
		router.NewMessageHandler(
			router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				return fimpgo.NewBoolMessage(
					fmt.Sprintf("evt.config.%s_report", SettingEnabled),
					tel.ServiceName(),
					tel.IsEnabled(),
					nil,
					nil,
					message.Payload,
				), nil
			})),
		router.ForService(tel.ServiceName()),
		router.ForType("cmd.config.get_"+SettingEnabled),
	)
}

func RouteCmdTelemetrySetEnabled(tel Telemetry) *router.Routing {
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
					fmt.Sprintf("evt.config.%s_report", SettingEnabled),
					tel.ServiceName(),
					enabled,
					nil,
					nil,
					message.Payload,
				), nil
			})),
		router.ForService(tel.ServiceName()),
		router.ForType("cmd.config.set_"+SettingEnabled),
	)
}

func RouteCmdTelemetryValidity(tel Telemetry) *router.Routing {
	return router.NewRouting(
		router.NewMessageHandler(
			router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				return fimpgo.NewStringMessage(
					fmt.Sprintf("evt.config.%s_report", SettingValidity),
					tel.ServiceName(),
					tel.Validity().String(),
					nil,
					nil,
					message.Payload,
				), nil
			})),
		router.ForService(tel.ServiceName()),
		router.ForType("cmd.config.get_"+SettingValidity),
	)
}

func RouteCmdTelemetrySetValidity(tel Telemetry) *router.Routing {
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
					fmt.Sprintf("evt.config.%s_report", SettingValidity),
					tel.ServiceName(),
					d.String(),
					nil,
					nil,
					message.Payload,
				), nil
			})),
		router.ForService(tel.ServiceName()),
		router.ForType("cmd.config.set_"+SettingValidity),
	)
}

func RouteCmdTelemetrySuppressed(tel Telemetry) *router.Routing {
	return router.NewRouting(
		router.NewMessageHandler(
			router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				return fimpgo.NewObjectMessage(
					fmt.Sprintf("evt.config.%s_report", SettingSuppressed),
					tel.ServiceName(),
					tel.Suppressed(),
					nil,
					nil,
					message.Payload,
				), nil
			})),
		router.ForService(tel.ServiceName()),
		router.ForType("cmd.config.get_"+SettingSuppressed),
	)
}

func RouteCmdTelemetrySetSuppressed(tel Telemetry) *router.Routing {
	return router.NewRouting(
		router.NewMessageHandler(
			router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
				var suppressed map[string]types.SuppressedEntry
				if err := message.Payload.GetObjectValue(&suppressed); err != nil {
					return nil, err
				}

				if err := tel.SetSuppressed(suppressed); err != nil {
					return nil, err
				}

				return fimpgo.NewObjectMessage(
					fmt.Sprintf("evt.config.%s_report", SettingSuppressed),
					tel.ServiceName(),
					tel.Suppressed(),
					nil,
					nil,
					message.Payload,
				), nil
			})),
		router.ForService(tel.ServiceName()),
		router.ForType("cmd.config.set_"+SettingSuppressed),
	)
}
