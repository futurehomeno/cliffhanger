package task_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestManager_Start(t *testing.T) {
	t.Parallel()

	type counterTaskMaker func(counter *uint) *task.Task

	makeCounterTask := func(dur time.Duration, voters ...task.Voter) counterTaskMaker {
		return func(counter *uint) *task.Task {
			return task.New(func() { *counter++ }, dur, voters...)
		}
	}

	skipVoter := task.VoterFn(func() bool { return false })
	passVoter := task.VoterFn(func() bool { return true })

	tests := []struct {
		name     string
		makers   []counterTaskMaker
		counters []uint
		sleep    time.Duration
	}{
		{
			name: "run single task once on startup",
			makers: []counterTaskMaker{
				makeCounterTask(0),
			},
			sleep:    10 * time.Millisecond,
			counters: []uint{1},
		},
		{
			name: "run single task once",
			makers: []counterTaskMaker{
				makeCounterTask(15 * time.Millisecond),
			},
			sleep:    10 * time.Millisecond,
			counters: []uint{1},
		},
		{
			name: "run single task twice",
			makers: []counterTaskMaker{
				makeCounterTask(10 * time.Millisecond),
			},
			counters: []uint{2},
			sleep:    15 * time.Millisecond,
		},
		{
			name: "run single task three times",
			makers: []counterTaskMaker{
				makeCounterTask(5 * time.Millisecond),
			},
			sleep:    12 * time.Millisecond,
			counters: []uint{3},
		},
		{
			name: "run two separate tasks",
			makers: []counterTaskMaker{
				makeCounterTask(10 * time.Millisecond),
				makeCounterTask(5 * time.Millisecond),
			},
			sleep:    12 * time.Millisecond,
			counters: []uint{2, 3},
		},
		{
			name: "run two separate tasks, always skip first one",
			makers: []counterTaskMaker{
				makeCounterTask(2*time.Millisecond, skipVoter),
				makeCounterTask(5*time.Millisecond, passVoter),
			},
			sleep:    12 * time.Millisecond,
			counters: []uint{0, 3},
		},
	}

	for _, ttt := range tests {
		tt := ttt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			counts := make([]uint, len(tt.makers))
			tasks := make([]*task.Task, len(tt.makers))

			for i, f := range tt.makers {
				tasks[i] = f(&counts[i])
			}

			manager := task.NewManager(tasks...)

			err := manager.Start()
			assert.NoError(t, err)

			time.Sleep(tt.sleep)

			err = manager.Stop()
			assert.NoError(t, err)

			for i, count := range counts {
				assert.Equal(t, tt.counters[i], count)
			}
		})
	}
}

func TestManager_Stop(t *testing.T) {
	t.Parallel()

	var functionFinished bool

	handler := func() {
		time.Sleep(15 * time.Millisecond)

		functionFinished = true
	}

	ts := task.New(handler, 5*time.Millisecond)

	r := task.NewManager(ts)

	err := r.Start()
	assert.NoError(t, err)

	time.Sleep(2 * time.Millisecond)

	err = r.Stop()
	assert.NoError(t, err)

	assert.True(t, functionFinished)
}

func TestManager(t *testing.T) {
	t.Parallel()

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:  "Test panic handling",
				Tasks: []*task.Task{task.New(func() { panic("test panic 1") }, 0)},
				Nodes: []*suite.Node{
					{
						Name:    "Execute task raising panic",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.test.test_command", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.test.test_event", "test_service", "test_value").Never(),
						},
						Timeout: 250 * time.Millisecond,
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestManager_StartStopErrorsAndRestart(t *testing.T) {
	t.Parallel()

	var runs atomic.Int32

	m := task.NewManager(task.New(func() { runs.Add(1) }, 0))

	assert.Error(t, m.Stop())
	assert.NoError(t, m.Start())
	assert.Error(t, m.Start())
	assert.NoError(t, m.Stop())
	assert.Error(t, m.Stop())
	assert.Equal(t, int32(1), runs.Load())

	assert.NoError(t, m.Start())
	assert.NoError(t, m.Stop())
	assert.Equal(t, int32(2), runs.Load(), "a run-once task runs again on every start")
}

func TestManager_PeriodicTaskSurvivesPanic(t *testing.T) {
	t.Parallel()

	var runs atomic.Int32

	m := task.NewManager(task.New(func() {
		if runs.Add(1) == 1 {
			panic("first tick")
		}
	}, 5*time.Millisecond))

	assert.NoError(t, m.Start())
	assert.Eventually(t, func() bool { return runs.Load() >= 2 }, time.Second, time.Millisecond)
	assert.NoError(t, m.Stop())
}

func TestManager_VoterReevaluatedOnEveryTick(t *testing.T) {
	t.Parallel()

	var (
		allowed atomic.Bool
		votes   atomic.Int32
		runs    atomic.Int32
	)

	voter := task.VoterFn(func() bool {
		votes.Add(1)

		return allowed.Load()
	})

	m := task.NewManager(task.New(func() { runs.Add(1) }, 2*time.Millisecond, voter))

	assert.NoError(t, m.Start())
	assert.Eventually(t, func() bool { return votes.Load() >= 3 }, time.Second, time.Millisecond)
	assert.Equal(t, int32(0), runs.Load())

	allowed.Store(true)

	assert.Eventually(t, func() bool { return runs.Load() >= 1 }, time.Second, time.Millisecond)
	assert.NoError(t, m.Stop())
}

func TestCombine(t *testing.T) {
	t.Parallel()

	a := task.New(func() {}, 0)
	b := task.New(func() {}, 0)
	c := task.New(func() {}, 0)

	assert.Equal(t, []*task.Task{a, b, c}, task.Combine([]*task.Task{a}, []*task.Task{b, c}))
	assert.Equal(t, []*task.Task{c, a}, task.Combine(c, a))
}
