package thing

import (
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/meterelec"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericsensor"
	"github.com/futurehomeno/cliffhanger/adapter/service/thermostat"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

// NewThermostat creates a thing that satisfies expectations for thermostat controller.
// Specification and implementations for temperature sensor and electricity meter are optional.
func NewThermostat(
	mqtt *fimpgo.MqttTransport,
	inclusionReport *fimptype.ThingInclusionReport,
	thermostatSpecification *fimptype.Service,
	thermostatController thermostat.ThermostatController,
	sensorTempSpecification *fimptype.Service,
	temperatureSensor numericsensor.NumericSensor,
	meterElecSpecification *fimptype.Service,
	electricityMeter meterelec.ElectricityMeter,
) adapter.Thing {
	services := []adapter.Service{
		thermostat.NewService(mqtt, thermostatSpecification, thermostatController),
	}

	if sensorTempSpecification != nil && temperatureSensor != nil {
		services = append(services, numericsensor.NewService(mqtt, sensorTempSpecification, temperatureSensor))
	}

	if meterElecSpecification != nil && electricityMeter != nil {
		services = append(services, meterelec.NewService(mqtt, meterElecSpecification, electricityMeter))
	}

	return adapter.NewThing(inclusionReport, services...)
}

// RouteThermostat creates routing required to satisfy expectations for thermostat controller.
func RouteThermostat(adapter adapter.Adapter) []*router.Routing {
	return router.Combine(
		thermostat.RouteService(adapter),
		numericsensor.RouteService(adapter),
		meterelec.RouteService(adapter),
	)
}

// TaskThermostat creates background tasks specific for thermostat controller.
func TaskThermostat(
	adapter adapter.Adapter,
	reportingInterval time.Duration,
	reportingVoters ...task.Voter,
) []*task.Task {
	return []*task.Task{
		numericsensor.TaskReporting(adapter, reportingInterval, reportingVoters...),
		meterelec.TaskReporting(adapter, reportingInterval, reportingVoters...),
	}
}
