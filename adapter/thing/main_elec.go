package thing

import (
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
	"github.com/futurehomeno/cliffhanger/adapter/service/meterelec"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

// MainElecConfig represents a thing configuration.
type MainElecConfig struct {
	InclusionReport *fimptype.ThingInclusionReport
	MeterElecConfig *meterelec.Config
}

// NewMainElec creates a thing that satisfies expectations for the main electricity meter.
func NewMainElec(
	mqtt *fimpgo.MqttTransport,
	cfg *MainElecConfig,
) adapter.Thing {
	if cfg.MeterElecConfig.ReportingStrategy == nil {
		cfg.MeterElecConfig.ReportingStrategy = cache.ReportAlways()
	}

	meterElec := meterelec.NewService(mqtt, cfg.MeterElecConfig)

	return adapter.NewThing(cfg.InclusionReport, meterElec)
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
