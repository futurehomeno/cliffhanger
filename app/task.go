package app

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/task"
)

// constants defining default periods for common application tasks.
const (
	defaultInitializationInterval = 5 * time.Minute
	defaultCheckInterval          = 30 * time.Minute
)

// TaskApp creates application tasks.
func TaskApp(app App, appLifecycle *lifecycle.Lifecycle) []*task.Task {
	var tasks []*task.Task

	initializable, ok := app.(InitializableApp)
	if ok {
		tasks = append(tasks, TaskInitialization(initializable, appLifecycle, defaultInitializationInterval)...)
	}

	checkable, ok := app.(CheckableApp)
	if ok {
		tasks = append(tasks, TaskCheck(checkable, appLifecycle, defaultCheckInterval))
	}

	return tasks
}

// TaskInitialization creates application initialization tasks.
func TaskInitialization(
	app InitializableApp,
	appLifecycle *lifecycle.Lifecycle,
	interval time.Duration,
) []*task.Task {
	handler := HandleInitialization(app, appLifecycle, interval)

	return []*task.Task{
		task.New(handler, 0),
		task.New(handler, interval, task.WhenAppEncounteredStartupError(appLifecycle)),
	}
}

// HandleInitialization creates handler of an initialization task.
func HandleInitialization(
	app InitializableApp,
	appLifecycle *lifecycle.Lifecycle,
	interval time.Duration,
) func() {
	return func() {
		err := app.Initialize()
		if err != nil {
			appLifecycle.SetAppState(lifecycle.AppStateStartupError, nil)
			log.WithError(err).Errorf("App init failed, retry in %s", interval)
		}
	}
}

// TaskCheck creates application check task.
func TaskCheck(
	app CheckableApp,
	appLifecycle *lifecycle.Lifecycle,
	interval time.Duration,
) *task.Task {
	handler := HandleCheck(app)

	return task.New(handler, interval, task.WhenAppIsRunning(appLifecycle))
}

// HandleCheck creates handler of a check task.
func HandleCheck(
	app CheckableApp,
) func() {
	return func() {
		err := app.Check()
		if err != nil {
			log.Errorf("Check app status err: %v", err)
		}
	}
}
