package task

import (
	"time"
)

// New creates new task. If duration is set to 0 the task will run only once on startup.
func New(handler func(), duration time.Duration, voters ...Voter) *Task {
	return &Task{
		handler:  handler,
		duration: duration,
		voters:   voters,
	}
}

// NewDelayed creates a periodic task whose first run waits for the first interval instead of running on startup.
// A zero duration has no interval to wait for, so the task runs once on startup like New.
func NewDelayed(handler func(), duration time.Duration, voters ...Voter) *Task {
	t := New(handler, duration, voters...)
	t.delayed = true

	return t
}

// Task is an object representing a task including its running interval and condition voters for being executed.
type Task struct {
	handler  func()
	duration time.Duration
	voters   []Voter
	delayed  bool
}

// run runs the task if all set conditions are met.
func (t *Task) run() {
	if !t.vote() {
		return
	}

	t.handler()
}

// vote checks if all set conditions are met by executing all registered voters.
func (t *Task) vote() bool {
	for _, v := range t.voters {
		if !v.Vote() {
			return false
		}
	}

	return true
}

// Combine is a helper to easily combine multiple instances or slices of tasks into one slice.
func Combine[T []*Task | *Task](parts ...T) []*Task {
	var combined []*Task

	for _, part := range parts {
		p, ok := any(part).(*Task)
		if ok {
			combined = append(combined, p)

			continue
		}

		ps, ok := any(part).([]*Task)
		if ok {
			combined = append(combined, ps...)

			continue
		}
	}

	return combined
}
