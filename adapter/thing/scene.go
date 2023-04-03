package thing

import (
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/scenectrl"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

// SceneConfig represents a config for a scene controller.
type SceneConfig struct {
	ThingConfig     *adapter.ThingConfig
	SceneCtrlConfig *scenectrl.Config
}

// NewScene creates scene controller thing.
func NewScene(
	publisher adapter.Publisher,
	ts adapter.ThingState,
	cfg *SceneConfig,
) adapter.Thing {
	services := []adapter.Service{
		scenectrl.NewService(publisher, cfg.SceneCtrlConfig),
	}

	return adapter.NewThing(publisher, ts, cfg.ThingConfig, services...)
}

// RouteScene creates routing required to satisfy expectations for the scene controller.
func RouteScene(ad adapter.Adapter) []*router.Routing {
	return router.Combine(
		scenectrl.RouteService(ad),
	)
}

// TaskScene creates background tasks specific for the scene controller.
func TaskScene(
	ad adapter.Adapter,
	interval time.Duration,
	voter ...task.Voter,
) []*task.Task {
	return []*task.Task{
		scenectrl.TaskReporting(ad, interval, voter...),
	}
}
