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

// NewThermostat creates a thing that satisfies expectations for a thermostat controller.
// Specification and implementations for temperature sensor and electricity meter are optional.
func NewThermostat(
	mqtt *fimpgo.MqttTransport,
	inclusionReport *fimptype.ThingInclusionReport,
	thermostatSpecification *fimptype.Service,
	thermostatController thermostat.Controller,
	sensorTempSpecification *fimptype.Service,
	sensorTempReporter numericsensor.Reporter,
	meterElecSpecification *fimptype.Service,
	meterElecReporter meterelec.Reporter,
) adapter.Thing {
	services := []adapter.Service{
		thermostat.NewService(mqtt, thermostatSpecification, thermostatController),
	}

	if sensorTempSpecification != nil && sensorTempReporter != nil && sensorTempSpecification.Name == numericsensor.SensorTemp {
		services = append(services, numericsensor.NewService(mqtt, sensorTempSpecification, sensorTempReporter))
	}

	if meterElecSpecification != nil && meterElecReporter != nil {
		services = append(services, meterelec.NewService(mqtt, meterElecSpecification, meterElecReporter))
	}

	return adapter.NewThing(inclusionReport, services...)
}

// RouteThermostat creates routing required to satisfy expectations for a thermostat controller.
func RouteThermostat(adapter adapter.Adapter) []*router.Routing {
	return router.Combine(
		thermostat.RouteService(adapter),
		numericsensor.RouteService(adapter),
		meterelec.RouteService(adapter),
	)
}

// TaskThermostat creates background tasks specific for a thermostat controller.
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
