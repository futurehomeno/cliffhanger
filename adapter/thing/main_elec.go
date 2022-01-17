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

func NewMainElec(
	inclusionReport *fimptype.ThingInclusionReport,
	meterElecSpecification *fimptype.Service,
	reporter meterelec.Reporter,
	extendedReporter meterelec.ExtendedReporter,
) adapter.Thing {
	meterElec := meterelec.NewService(meterElecSpecification, reporter, extendedReporter)

	return adapter.NewThing(inclusionReport, meterElec)
}

func RouteMainElec(adapter adapter.Adapter) []*router.Routing {
	return meterelec.RouteService(adapter)
}

func TaskMainElec(
	adapter adapter.Adapter,
	mqtt *fimpgo.MqttTransport,
	reportingDuration time.Duration,
	reportingVoters ...task.Voter,
) []*task.Task {
	return []*task.Task{
		meterelec.TaskReporting(adapter, mqtt, reportingDuration, reportingVoters...),
	}
}
