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

type Service interface {
	Start() error
	Stop() error
}

type Callback func(t *testing.T)

func NewCase(name string) *Case {
	return &Case{
		Name: name,
	}
}

type Case struct {
	Name     string
	Routing  []*router.Routing
	Tasks    []*task.Task
	Service  Service
	Mocks    []Mock
	Setup    Setup
	TearDown []Callback
	Nodes    []*Node
}

func (c *Case) WithName(name string) *Case {
	c.Name = name

	return c
}

func (c *Case) WithRouting(routing ...*router.Routing) *Case {
	c.Routing = append(c.Routing, routing...)

	return c
}

func (c *Case) WithTasks(tasks ...*task.Task) *Case {
	c.Tasks = append(c.Tasks, tasks...)

	return c
}

func (c *Case) WithService(service Service) *Case {
	c.Service = service

	return c
}

func (c *Case) WithMocks(mocks ...Mock) *Case {
	c.Mocks = append(c.Mocks, mocks...)

	return c
}

func (c *Case) WithSetup(setup Setup) *Case {
	c.Setup = setup

	return c
}

func (c *Case) WithNodes(nodes ...*Node) *Case {
	c.Nodes = append(c.Nodes, nodes...)

	return c
}

func (c *Case) WithTearDown(callbacks ...Callback) *Case {
	c.TearDown = append(c.TearDown, callbacks...)

	return c
}

func (c *Case) Run(t *testing.T, mqtt *fimpgo.MqttTransport) {
	t.Helper()

	c.init(t, mqtt)
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

func (c *Case) init(t *testing.T, mqtt *fimpgo.MqttTransport) {
	t.Helper()

	if c.Setup != nil {
		c.Setup.apply(t, mqtt, c)
	}

	if len(c.Nodes) == 0 {
		return
	}

	c.initRouting(mqtt)
	c.initTasks()
	c.initService()
}

func (c *Case) initRouting(mqtt *fimpgo.MqttTransport) {
	if len(c.Routing) == 0 {
		return
	}

	r := router.NewRouter(mqtt, "cliffhanger_test_case", c.Routing...)

	initCallback := func(t *testing.T) {
		err := r.Start()
		if err != nil {
			t.Fatalf("failed to start the router for the test case: %s", err)
		}
	}

	c.Nodes[0].InitCallbacks = append(c.Nodes[0].InitCallbacks, initCallback)

	tearDownCallback := func(t *testing.T) {
		err := r.Stop()
		if err != nil {
			t.Fatalf("failed to stop the router for the test case: %s", err)
		}
	}

	c.TearDown = append(c.TearDown, tearDownCallback)
}

func (c *Case) initTasks() {
	if len(c.Tasks) == 0 {
		return
	}

	taskManager := task.NewManager(c.Tasks...)

	initCallback := func(t *testing.T) {
		err := taskManager.Start()
		if err != nil {
			t.Fatalf("failed to start the task manager for the test case: %s", err)
		}
	}

	c.Nodes[0].InitCallbacks = append(c.Nodes[0].InitCallbacks, initCallback)

	tearDownCallback := func(t *testing.T) {
		err := taskManager.Stop()
		if err != nil {
			t.Fatalf("failed to stop the task manager for the test case: %s", err)
		}
	}

	c.TearDown = append(c.TearDown, tearDownCallback)
}

func (c *Case) initService() {
	if c.Service == nil {
		return
	}

	initCallback := func(t *testing.T) {
		err := c.Service.Start()
		if err != nil {
			t.Fatalf("failed to start the service for the test case: %s", err)
		}
	}

	c.Nodes[0].InitCallbacks = append(c.Nodes[0].InitCallbacks, initCallback)

	tearDownCallback := func(t *testing.T) {
		err := c.Service.Stop()
		if err != nil {
			t.Fatalf("failed to stop the service for the test case: %s", err)
		}
	}

	c.TearDown = append(c.TearDown, tearDownCallback)
}

func (c *Case) tearDown(t *testing.T) {
	t.Helper()

	for _, td := range c.TearDown {
		td(t)
	}
}
