package suite

import (
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/mock"

	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

type Mock interface {
	AssertExpectations(t mock.TestingT) bool
}

type CaseSetup func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []Mock)

func NewCase(name string) *Case {
	return &Case{
		Name: name,
	}
}

type Case struct {
	Name    string
	Routing []*router.Routing
	Tasks   []*task.Task
	Mocks   []Mock
	Setup   CaseSetup
	Nodes   []*Node

	router      router.Router
	taskManager task.Manager
}

func (c *Case) WithName(name string) *Case {
	c.Name = name

	return c
}

func (c *Case) WithNodes(nodes ...*Node) *Case {
	c.Nodes = append(c.Nodes, nodes...)

	return c
}

func (c *Case) WithRouting(routing ...*router.Routing) *Case {
	c.Routing = append(c.Routing, routing...)

	return c
}

func (c *Case) Run(t *testing.T, mqtt *fimpgo.MqttTransport) {
	t.Helper()

	c.setup(t, mqtt)
	defer c.tearDown(t)

	for _, tn := range c.Nodes {
		n := tn

		t.Run(n.Name, func(t *testing.T) {
			t.Helper()

			n.Run(t, mqtt)
		})
	}

	for _, m := range c.Mocks {
		m.AssertExpectations(t)
	}
}

func (c *Case) setup(t *testing.T, mqtt *fimpgo.MqttTransport) {
	t.Helper()

	if c.Setup != nil {
		routing, tasks, mocks := c.Setup(t, mqtt)

		c.Routing = append(c.Routing, routing...)
		c.Tasks = append(c.Tasks, tasks...)
		c.Mocks = append(c.Mocks, mocks...)
	}

	if len(c.Routing) > 0 {
		c.router = router.NewRouter(mqtt, "cliffhanger_test_case", c.Routing...)

		err := c.router.Start()
		if err != nil {
			t.Fatalf("failed to start the router for the test case: %s", err)
		}
	}

	if len(c.Tasks) > 0 && len(c.Nodes) > 0 {
		c.taskManager = task.NewManager(c.Tasks...)

		c.Nodes[0].postExpectationCallback = func(t *testing.T) {
			err := c.taskManager.Start()
			if err != nil {
				t.Fatalf("failed to start the task manager for the test case: %s", err)
			}
		}
	}
}

func (c *Case) tearDown(t *testing.T) {
	t.Helper()

	if c.router != nil {
		err := c.router.Stop()
		if err != nil {
			t.Fatalf("failed to stop the router for the test case: %s", err)
		}

		c.router = nil
	}

	if c.taskManager != nil && len(c.Nodes) > 0 {
		err := c.taskManager.Stop()
		if err != nil {
			t.Fatalf("failed to stop the task manager for the test case: %s", err)
		}

		c.taskManager = nil
	}
}
