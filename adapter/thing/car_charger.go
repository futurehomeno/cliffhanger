package thing

import (
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/meterelec"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericsensor"
	"github.com/futurehomeno/cliffhanger/adapter/service/waterheater"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

// NewCarCharger creates a thing that satisfies expectations for a car charger.
// Specification and implementation for electricity meter is optional.
func NewCarCharger(
	mqtt *fimpgo.MqttTransport,
	inclusionReport *fimptype.ThingInclusionReport,
	chargepointSpecification *fimptype.Service,
	chargepointController chargepoint.Controller,
	meterElecSpecification *fimptype.Service,
	meterElecReporter meterelec.Reporter,
) adapter.Thing {
	services := []adapter.Service{
		chargepoint.NewService(mqtt, chargepointSpecification, chargepointController),
	}

	if meterElecSpecification != nil && meterElecReporter != nil {
		services = append(services, meterelec.NewService(mqtt, meterElecSpecification, meterElecReporter))
	}

	return adapter.NewThing(inclusionReport, services...)
}

// RouteCarCharger creates routing required to satisfy expectations for a car charger.
func RouteCarCharger(adapter adapter.Adapter) []*router.Routing {
	return router.Combine(
		waterheater.RouteService(adapter),
		numericsensor.RouteService(adapter),
		meterelec.RouteService(adapter),
	)
}

// TaskCarCharger creates background tasks specific for a car charger.
func TaskCarCharger(
	adapter adapter.Adapter,
	reportingInterval time.Duration,
	reportingVoters ...task.Voter,
) []*task.Task {
	return []*task.Task{
		meterelec.TaskReporting(adapter, reportingInterval, reportingVoters...),
	}
}
