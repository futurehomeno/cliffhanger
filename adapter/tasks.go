package adapter

import (
	"time"

	"github.com/futurehomeno/cliffhanger/task"
)

// TaskAdapter creates background tasks specific for an adapter.
func TaskAdapter(
	adapter Adapter,
	reportingInterval time.Duration,
	reportingVoters ...task.Voter,
) []*task.Task {
	return []*task.Task{
		// TODO: Add tasks for connectivity reporting and maybe periodic checks of changes in the inclusion report.
	}
}
