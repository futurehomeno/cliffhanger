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
	waterHeaterController waterheater.Controller,
	sensorWatTempSpecification *fimptype.Service,
	sensorWatTempReporter numericsensor.Reporter,
	meterElecSpecification *fimptype.Service,
	meterElecReporter meterelec.Reporter,
) adapter.Thing {
	services := []adapter.Service{
		waterheater.NewService(mqtt, waterHeaterSpecification, waterHeaterController),
	}

	if sensorWatTempSpecification != nil && sensorWatTempReporter != nil && sensorWatTempSpecification.Name == numericsensor.SensorWatTemp {
		services = append(services, numericsensor.NewService(mqtt, sensorWatTempSpecification, sensorWatTempReporter))
	}

	if meterElecSpecification != nil && meterElecReporter != nil {
		services = append(services, meterelec.NewService(mqtt, meterElecSpecification, meterElecReporter))
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
