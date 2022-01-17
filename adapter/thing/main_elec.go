package mainelec

import (
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/meterelec"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

// NewMainElec creates a thing that satisfies expectations for main electricity meter.
func NewMainElec(
	inclusionReport *fimptype.ThingInclusionReport,
	meterElecSpecification *fimptype.Service,
	meterElecReporter meterelec.Reporter,
) adapter.Thing {
	meterElec := meterelec.NewService(meterElecSpecification, meterElecReporter)

	return adapter.NewThing(inclusionReport, meterElec)
}

// RouteMainElec creates routing required to satisfy expectations for main electricity meter.
func RouteMainElec(adapter adapter.Adapter) []*router.Routing {
	return meterelec.RouteService(adapter)
}

// TaskMainElec creates background tasks specific to main electricity meter.
func TaskMainElec(
	adapter adapter.Adapter,
	mqtt *fimpgo.MqttTransport,
	reportingInterval time.Duration,
	reportingVoters ...task.Voter,
) []*task.Task {
	return []*task.Task{
		meterelec.TaskReporting(adapter, mqtt, reportingInterval, reportingVoters...),
	}
}
