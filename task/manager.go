package task

import (
	"errors"
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/utils"
	log "github.com/sirupsen/logrus"
)

// Manager is an interface representing a tasks manager service.
type Manager interface {
	Start() error // Start starts the manager and all its tasks.
	Stop() error  // Stop stops the manager and all its tasks.
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
		return errors.New("task already running")
	}

	r.stopCh = make(chan struct{})

	r.wg.Add(len(r.tasks))

	for _, task := range r.tasks {
		if task.interval == 0 {
			go runOnce(task, r.wg)
			continue
		}

		go runPeriodically(task, r.stopCh, r.wg)
	}

	return nil
}

// Stop stops the manager and all its tasks.
func (r *manager) Stop() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.stopCh == nil {
		return errors.New("task not running")
	}

	close(r.stopCh)
	r.wg.Wait()

	r.stopCh = nil

	return nil
}

// runOnce runs the task once if it's running interval is set to 0.
func runOnce(task *Task, wg *sync.WaitGroup) {
	run(task)
	wg.Done()
}

// runPeriodically runs the task according to the provided interval.
func runPeriodically(task *Task, stopC chan struct{}, wg *sync.WaitGroup) {
	defer utils.PrintStackOnRecover(true, "runPeriodically")
	ticker := time.NewTicker(task.interval)
	defer ticker.Stop()

	run(task)

	for {
		select {
		case <-ticker.C:
			run(task)

		case <-stopC:
			wg.Done()
			return
		}
	}
}

// run executes the task with a panic recovery.
func run(task *Task) {
	defer utils.PrintStackOnRecover(false, "task")

	if task == nil {
		log.Error("[cliff] Task is nil")
		return
	}

	task.run()
}
