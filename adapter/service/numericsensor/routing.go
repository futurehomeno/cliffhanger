package numericsensor

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing service, commands and events.
const (
	CmdSensorGetReport = "cmd.sensor.get_report"
	EvtSensorReport    = "evt.sensor.report"

	SensorAccelX      = prefix + "accelx"
	SensorAccelY      = prefix + "accely"
	SensorAccelZ      = prefix + "accelz"
	SensorAirflow     = prefix + "airflow"
	SensorAnglePos    = prefix + "anglepos"
	SensorAtmo        = prefix + "atmo"
	SensorBaro        = prefix + "baro"
	SensorCO2         = prefix + "co2"
	SensorCO          = prefix + "co"
	SensorCurrent     = prefix + "current"
	SensorDew         = prefix + "dew"
	SensorDirect      = prefix + "direct"
	SensorDistance    = prefix + "distance"
	SensorElResist    = prefix + "elresist"
	SensorFreq        = prefix + "freq"
	SensorGP          = prefix + "gp"
	SensorGust        = prefix + "gust"
	SensorHumid       = prefix + "humid"
	SensorLumin       = prefix + "lumin"
	SensorMoist       = prefix + "moist"
	SensorNoise       = prefix + "noise"
	SensorPower       = prefix + "power"
	SensorRain        = prefix + "rain"
	SensorRotation    = prefix + "rotation"
	SensorSeismicInt  = prefix + "seismicint"
	SensorSeismicMag  = prefix + "seismicmag"
	SensorSolarRad    = prefix + "solarrad"
	SensorTank        = prefix + "tank"
	SensorTemp        = prefix + "temp"
	SensorTideLvl     = prefix + "tidelvl"
	SensorUV          = prefix + "uv"
	SensorVeloc       = prefix + "veloc"
	SensorVoltage     = prefix + "voltage"
	SensorWatFlow     = prefix + "watflow"
	SensorWatPressure = prefix + "watpressure"
	SensorWatTemp     = prefix + "wattemp"
	SensorWeight      = prefix + "weight"
	SensorWind        = prefix + "wind"

	prefix = "sensor_"
)

// RouteService returns routing for service specific commands.
func RouteService(serviceRegistry adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		RouteCmdSensorGetReport(serviceRegistry),
	}
}

// RouteCmdSensorGetReport returns a routing responsible for handling the command.
func RouteCmdSensorGetReport(serviceRegistry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		HandleCmdSensorGetReport(serviceRegistry),
		router.ForServicePrefix(prefix),
		router.ForType(CmdSensorGetReport),
	)
}

// HandleCmdSensorGetReport returns a handler responsible for handling the command.
func HandleCmdSensorGetReport(serviceRegistry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (reply *fimpgo.FimpMessage, err error) {
			s := serviceRegistry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			numericSensor, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			if numericSensor.Name() != message.Payload.Service {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			units, err := unitsToReport(numericSensor, message)
			if err != nil {
				return nil, err
			}

			for _, unit := range units {
				_, err = numericSensor.SendSensorReport(unit, true)
				if err != nil {
					return nil, fmt.Errorf("adapter: failed to send sensor report: %w", err)
				}
			}

			return nil, nil
		}),
	)
}

// unitsToReport is a helper method that determines which units should be reported.
func unitsToReport(service Service, message *fimpgo.Message) ([]string, error) {
	if message.Payload.ValueType == fimpgo.VTypeNull {
		return service.SupportedUnits(), nil
	}

	unit, err := message.Payload.GetStringValue()
	if err != nil {
		return nil, fmt.Errorf("adapter: provided unit has an incorrect format: %w", err)
	}

	if unit == "" {
		return service.SupportedUnits(), nil
	}

	return []string{unit}, nil
}
