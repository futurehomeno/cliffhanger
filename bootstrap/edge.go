package bootstrap

import (
	"time"

	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/storage"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/cliffhanger/telemetry"
)

// EdgeRouting combines the standard edge adapter routing (default, app and adapter routes)
// with domain-specific extras. excludeAllThings is forwarded to app.RouteApp: typically
// ad.DestroyAllThings, or nil when the application destroys things in Uninstall itself.
// adapterOptions is forwarded to adapter.RouteAdapter: typically adapter.WithSelectionRemover
// and adapter.WithLocker(locker) for an application with a device selection.
func EdgeRouting[C any](
	serviceName fimptype.ServiceNameT,
	configGetter func() any,
	tel telemetry.Telemetry,
	appLifecycle *lifecycle.Lifecycle,
	configStorage storage.Storage[C],
	configFactory func() C,
	locker router.MessageHandlerLocker,
	application app.App,
	ad adapter.Adapter,
	excludeAllThings func() error,
	adapterOptions []adapter.RoutingOption,
	extras ...[]*router.Routing,
) []*router.Routing {
	routes := [][]*router.Routing{
		DefaultRoute(serviceName, configGetter, tel),
		app.RouteApp(serviceName, appLifecycle, configStorage, configFactory, locker, application, excludeAllThings),
		adapter.RouteAdapter(ad, adapterOptions...),
	}

	return router.Combine(append(routes, extras...)...)
}

// EdgeTasks combines the standard edge adapter tasks (app and adapter tasks)
// with domain-specific extras.
func EdgeTasks(
	application app.App,
	appLifecycle *lifecycle.Lifecycle,
	ad adapter.Adapter,
	reportingInterval time.Duration,
	extras ...[]*task.Task,
) []*task.Task {
	tasks := [][]*task.Task{
		app.TaskApp(application, appLifecycle),
		adapter.TaskAdapter(ad, reportingInterval),
	}

	return task.Combine(append(tasks, extras...)...)
}
