package battery

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing service, commands and events.
const (
	CmdLevelGetReport   = "cmd.lvl.get_report"
	EvtLevelReport      = "evt.lvl.report"
	EvtAlarmReport      = "evt.alarm.report"
	CmdHealthGetReport  = "cmd.health.get_report"
	EvtHealthReport     = "evt.health.report"
	CmdSensorGetReport  = "cmd.sensor.get_report"
	EvtSensorReport     = "evt.sensor.report"
	CmdBatteryGetReport = "cmd.battery.get_report"
	EvtBatteryReport    = "evt.battery.report"

	Battery = "battery"
)

// RouteService returns routing for service specific commands.
func RouteService(adapter adapter.Adapter) []*router.Routing {
	return []*router.Routing{
		RouteCmdLevelGetReport(adapter),
		RouteCmdHealthGetReport(adapter),
		RouteCmdSensorGetReport(adapter),
		RouteCmdBatteryGetReport(adapter),
	}
}

// RouteCmdLevelGetReport returns a routing responsible for handling the command.
func RouteCmdLevelGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdLevelGetReport(adapter),
		router.ForService(Battery),
		router.ForType(CmdLevelGetReport),
	)
}

// HandleCmdLevelGetReport returns a handler responsible for handling the command.
func HandleCmdLevelGetReport(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			battery, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := battery.SendBatteryLevelReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send battery level report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdHealthGetReport returns a routing responsible for handling the command.
func RouteCmdHealthGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdHealthGetReport(adapter),
		router.ForService(Battery),
		router.ForType(CmdHealthGetReport),
	)
}

// HandleCmdHealthGetReport returns a handler responsible for handling the command.
func HandleCmdHealthGetReport(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			battery, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err = battery.SendBatteryHealthReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send battery health report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdSensorGetReport returns a routing responsible for handling the command.
func RouteCmdSensorGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdSensorGetReport(adapter),
		router.ForService(Battery),
		router.ForType(CmdSensorGetReport),
	)
}

// HandleCmdSensorGetReport returns a handler responsible for handling the command.
func HandleCmdSensorGetReport(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			battery, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err = battery.SendBatterySensorReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send battery sensor report: %w", err)
			}

			return nil, nil
		}),
	)
}

// RouteCmdBatteryGetReport returns a routing responsible for handling the command.
func RouteCmdBatteryGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdBatteryGetReport(adapter),
		router.ForService(Battery),
		router.ForType(CmdBatteryGetReport),
	)
}

// HandleCmdBatteryGetReport returns a handler responsible for handling the command.
func HandleCmdBatteryGetReport(adapter adapter.Adapter) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			s := adapter.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			battery, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err = battery.SendBatteryFullReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send battery battery report: %w", err)
			}

			return nil, nil
		}),
	)
}
