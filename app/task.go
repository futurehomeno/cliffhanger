package app

import (
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/task"
)

const (
	defaultAppInitInterval = 10 * time.Minute
	defaultCheckInterval   = 30 * time.Minute
)

func TaskApp(app App, appLifecycle *lifecycle.Lifecycle) []*task.Task {
	var tasks []*task.Task

	if initializable, ok := app.(InitializableApp); ok {
		tasks = append(tasks, TaskInitialization(initializable, appLifecycle, defaultAppInitInterval)...)
	}

	if checkable, ok := app.(CheckableApp); ok {
		interval := checkable.CheckInterval()
		if interval == 0 {
			interval = defaultCheckInterval
		}

		tasks = append(tasks, TaskCheck(checkable, appLifecycle, interval))
	}

	return tasks
}

func TaskInitialization(
	app InitializableApp,
	appLifecycle *lifecycle.Lifecycle,
	interval time.Duration,
) []*task.Task {
	handler := HandleInitialization(app, appLifecycle, interval)

	var started atomic.Bool

	// The periodic task runs once immediately; skip it so it cannot double the one-shot startup run.
	skipFirstRun := task.VoterFn(func() bool { return started.Swap(true) })

	return []*task.Task{
		task.New(handler, 0),
		task.New(handler, interval, skipFirstRun, task.WhenAppEncounteredStartupError(appLifecycle)),
	}
}

func HandleInitialization(
	app InitializableApp,
	appLifecycle *lifecycle.Lifecycle,
	interval time.Duration,
) func() {
	return func() {
		err := app.Initialize()
		if err != nil {
			appLifecycle.SetAppHealth(lifecycle.AppHealthStartupError, nil)
			log.Errorf("[app] Initialize app err: %v, retry in %s", err, interval)
		}
	}
}

func TaskCheck(
	app CheckableApp,
	appLifecycle *lifecycle.Lifecycle,
	interval time.Duration,
) *task.Task {
	handler := HandleCheck(app)

	return task.New(handler, interval, task.WhenAppIsRunning(appLifecycle))
}

func HandleCheck(
	app CheckableApp,
) func() {
	return func() {
		err := app.Check()
		if err != nil {
			log.Errorf("[app] Check app status err: %v", err)
		}
	}
}
