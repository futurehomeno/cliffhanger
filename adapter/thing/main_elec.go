package thing

import (
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/meterelec"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

// NewMainElec creates a thing that satisfies expectations for the main electricity meter.
func NewMainElec(
	mqtt *fimpgo.MqttTransport,
	inclusionReport *fimptype.ThingInclusionReport,
	meterElecSpecification *fimptype.Service,
	electricityMeter meterelec.ElectricityMeter,
) adapter.Thing {
	meterElec := meterelec.NewService(mqtt, meterElecSpecification, electricityMeter)

	return adapter.NewThing(inclusionReport, meterElec)
}

// RouteMainElec creates routing required to satisfy expectations for the main electricity meter.
func RouteMainElec(adapter adapter.Adapter) []*router.Routing {
	return meterelec.RouteService(adapter)
}

// TaskMainElec creates background tasks specific for the main electricity meter.
func TaskMainElec(
	adapter adapter.Adapter,
	reportingInterval time.Duration,
	reportingVoters ...task.Voter,
) []*task.Task {
	return []*task.Task{
		meterelec.TaskReporting(adapter, reportingInterval, reportingVoters...),
	}
}
