package task

import (
	"runtime/debug"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// Manager is an interface representing a tasks manager service.
type Manager interface {
	// Start starts the manager and all its tasks.
	Start() error
	// Stop stops the manager and all its tasks.
	Stop() error
	// UpdateTaskInterval updates a task with a new duration
	UpdateTaskInterval(name string, duration time.Duration) error
}

// manager is the implementation of the task manager interface.
type manager struct {
	tasks []*Task
	wg    *sync.WaitGroup
	lock  *sync.Mutex
}

// NewManager returns a new task manager.
func NewManager(tasks ...*Task) Manager {
	return &manager{
		wg:    &sync.WaitGroup{},
		tasks: tasks,
		lock:  &sync.Mutex{},
	}
}

// Start starts the manager and all its tasks.
func (r *manager) Start() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	for _, task := range r.tasks {
		t := task
		r.startTask(t)
	}

	return nil
}

// Stop stops the manager and all its tasks.
func (r *manager) Stop() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	for _, task := range r.tasks {
		if task.stopCh != nil {
			close(task.stopCh)
			task.stopCh = nil
		}
	}

	r.wg.Wait()

	return nil
}

// UpdateTaskInterval updates a named task with a provided duration.
func (r *manager) UpdateTaskInterval(name string, duration time.Duration) error {
	if name == Anonymous {
		return nil
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	for _, t := range r.tasks {
		if t.name == name {
			log.Infof("Updating task %s, with new duration: %v", name, duration)

			t.duration = duration
			r.restart(t)

			break
		}
	}

	return nil
}

// startTask starts the flow of running task once or continuously depending on the duration.
func (r *manager) startTask(task *Task) {
	task.stopCh = make(chan struct{})

	r.wg.Add(1)

	go func() {
		defer r.wg.Done()

		r.run(task)

		if task.duration != 0 {
			r.runContinuously(task)
		}
	}()
}

// runContinuously runs the task according to the provided interval.
func (r *manager) runContinuously(task *Task) {
	ticker := time.NewTicker(task.duration)
	defer ticker.Stop()

	stopCh := task.stopCh

	for {
		select {
		case <-ticker.C:
			r.run(task)

		case <-stopCh:
			return
		}
	}
}

// restart stops the current task and starts it again with a new stop channel.
func (r *manager) restart(task *Task) {
	if task.stopCh != nil {
		close(task.stopCh)
	}

	r.startTask(task)
}

// run executes the task with a panic recovery.
func (r *manager) run(task *Task) {
	defer func() {
		if r := recover(); r != nil {
			log.WithField("stack", string(debug.Stack())).
				Errorf("task manager: panic occurred while running a task: %+v", r)
		}
	}()

	task.run()
}
