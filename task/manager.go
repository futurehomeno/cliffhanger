package task

import (
	"errors"
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
}

// manager is the implementation of the task manager interface.
type manager struct {
	tasks  []*Task
	stopCh chan struct{}
	wg     *sync.WaitGroup
	lock   *sync.Mutex
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

	if r.stopCh != nil {
		return errors.New("task manager: cannot be started as it is already running")
	}

	r.stopCh = make(chan struct{})

	r.wg.Add(len(r.tasks))

	for _, task := range r.tasks {
		if task.duration == 0 {
			go r.runOnce(task)

			continue
		}

		go r.runContinuously(task)
	}

	return nil
}

// Stop stops the manager and all its tasks.
func (r *manager) Stop() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.stopCh == nil {
		return errors.New("task manager: cannot be stopped as it is already not running")
	}

	close(r.stopCh)
	r.wg.Wait()

	r.stopCh = nil

	return nil
}

// runOnce runs the task once if it's running interval is set to 0.
func (r *manager) runOnce(task *Task) {
	r.run(task)
	r.wg.Done()
}

// runContinuously runs the task according to the provided interval.
func (r *manager) runContinuously(task *Task) {
	ticker := time.NewTicker(task.duration)
	defer ticker.Stop()

	r.run(task)

	for {
		select {
		case <-ticker.C:
			r.run(task)

		case <-r.stopCh:
			r.wg.Done()

			return
		}
	}
}

// run executes the task with a panic recovery.
func (r *manager) run(task *Task) {
	defer func() {
		if r := recover(); r != nil {
			log.WithField("stack", string(debug.Stack())).
				Errorf("task manager: panic occurred while running a task: %+v", r)
		}
	}()

	if task == nil {
		log.Error("[cliff] Task is nil")
		return
	}

	task.run()
}
