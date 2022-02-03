package sensornumeric

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

	SensorAccelX      = "sensor_accelx"
	SensorAccelY      = "sensor_accely"
	SensorAccelZ      = "sensor_accelz"
	SensorAirflow     = "sensor_airflow"
	SensorAnglePos    = "sensor_anglepos"
	SensorAtmo        = "sensor_atmo"
	SensorBaro        = "sensor_baro"
	SensorCO2         = "sensor_co2"
	SensorCO          = "sensor_co"
	SensorCurrent     = "sensor_current"
	SensorDew         = "sensor_dew"
	SensorDirect      = "sensor_direct"
	SensorDistance    = "sensor_distance"
	SensorElResist    = "sensor_elresist"
	SensorFreq        = "sensor_freq"
	SensorGP          = "sensor_gp"
	SensorGust        = "sensor_gust"
	SensorHumid       = "sensor_humid"
	SensorLumin       = "sensor_lumin"
	SensorMoist       = "sensor_moist"
	SensorNoise       = "sensor_noise"
	SensorPower       = "sensor_power"
	SensorRain        = "sensor_rain"
	SensorRotation    = "sensor_rotation"
	SensorSeismicInt  = "sensor_seismicint"
	SensorSeismicMag  = "sensor_seismicmag"
	SensorSolarRad    = "sensor_solarrad"
	SensorTank        = "sensor_tank"
	SensorTemp        = "sensor_temp"
	SensorTideLvl     = "sensor_tidelvl"
	SensorUV          = "sensor_uv"
	SensorVeloc       = "sensor_veloc"
	SensorVoltage     = "sensor_voltage"
	SensorWatFlow     = "sensor_watflow"
	SensorWatPressure = "sensor_watpressure"
	SensorWatTemp     = "sensor_wattemp"
	SensorWeight      = "sensor_weight"
	SensorWind        = "sensor_wind"
)

// RouteService returns routing for service specific commands.
func RouteService(adapter adapter.Adapter) []*router.Routing {
	return []*router.Routing{
		RouteCmdSensorGetReport(adapter),
	}
}

// RouteCmdSensorGetReport returns a routing responsible for handling the command.
func RouteCmdSensorGetReport(adapter adapter.Adapter) *router.Routing {
	return router.NewRouting(
		HandleCmdSensorGetReport(adapter),
		router.ForServicePrefix("sensor_"),
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

			sensor, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			if sensor.Name() != message.Payload.Service {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			if message.Payload.ValueType != fimpgo.VTypeString && message.Payload.ValueType != fimpgo.VTypeNull {
				return nil, fmt.Errorf(
					"adapter: provided message value has an invalid type, received %s instead of %s or %s",
					message.Payload.ValueType, fimpgo.VTypeString, fimpgo.VTypeNull,
				)
			}

			var units []string
			if message.Payload.ValueType == fimpgo.VTypeString {
				var unit string
				unit, err = message.Payload.GetStringValue()
				if err != nil {
					return nil, fmt.Errorf("adapter: provided unit has an incorrect format: %w", err)
				}

				units = append(units, unit)
			} else {
				units = sensor.SupportedUnits()
			}

			for _, unit := range units {
				_, err = sensor.SendReport(unit, true)
				if err != nil {
					return nil, fmt.Errorf("adapter: failed to send report: %w", err)
				}
			}

			return nil, nil
		}),
	)
}
