package thing

import (
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/meterelec"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericsensor"
	"github.com/futurehomeno/cliffhanger/adapter/service/waterheater"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

// NewBoiler creates a thing that satisfies expectations for a boiler.
// Specification and implementations for temperature sensor and electricity meter are optional.
func NewBoiler(
	mqtt *fimpgo.MqttTransport,
	inclusionReport *fimptype.ThingInclusionReport,
	waterHeaterSpecification *fimptype.Service,
	waterHeaterController waterheater.WaterHeaterController,
	sensorTempSpecification *fimptype.Service,
	temperatureSensor numericsensor.NumericSensor,
	meterElecSpecification *fimptype.Service,
	electricityMeter meterelec.ElectricityMeter,
) adapter.Thing {
	services := []adapter.Service{
		waterheater.NewService(mqtt, waterHeaterSpecification, waterHeaterController),
	}

	if sensorTempSpecification != nil && temperatureSensor != nil {
		services = append(services, numericsensor.NewService(mqtt, sensorTempSpecification, temperatureSensor))
	}

	if meterElecSpecification != nil && electricityMeter != nil {
		services = append(services, meterelec.NewService(mqtt, meterElecSpecification, electricityMeter))
	}

	return adapter.NewThing(inclusionReport, services...)
}

// RouteBoiler creates routing required to satisfy expectations for a boiler.
func RouteBoiler(adapter adapter.Adapter) []*router.Routing {
	return router.Combine(
		waterheater.RouteService(adapter),
		numericsensor.RouteService(adapter),
		meterelec.RouteService(adapter),
	)
}

// TaskBoiler creates background tasks specific for a boiler.
func TaskBoiler(
	adapter adapter.Adapter,
	reportingInterval time.Duration,
	reportingVoters ...task.Voter,
) []*task.Task {
	return []*task.Task{
		numericsensor.TaskReporting(adapter, reportingInterval, reportingVoters...),
		meterelec.TaskReporting(adapter, reportingInterval, reportingVoters...),
	}
}
