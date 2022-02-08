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

// Task is an object representing a task including its running interval and condition voters for being executed.
type Task struct {
	handler  func()
	duration time.Duration
	voters   []Voter
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

// Combine is a helper to easily combine multiple slices of tasks into one.
func Combine(parts ...[]*Task) []*Task {
	var combined []*Task

	for _, p := range parts {
		combined = append(combined, p...)
	}

	return combined
}
