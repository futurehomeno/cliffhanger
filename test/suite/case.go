package suite

import (
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/mock"

	"github.com/futurehomeno/cliffhanger/router"
)

type Mock interface {
	AssertExpectations(t mock.TestingT) bool
}

type CaseSetup func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []Mock)

func NewCase(name string) *Case {
	return &Case{
		Name: name,
	}
}

type Case struct {
	Name    string
	Routing []*router.Routing
	Mocks   []Mock
	Setup   CaseSetup
	Nodes   []*Node

	router router.Router
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
		routing, mocks := c.Setup(t, mqtt)

		c.Routing = append(c.Routing, routing...)
		c.Mocks = append(c.Mocks, mocks...)
	}

	if len(c.Routing) > 0 {
		c.router = router.NewRouter(mqtt, "cliffhanger_test_case", c.Routing...)

		err := c.router.Start()
		if err != nil {
			t.Fatalf("failed to start the router for the test case: %s", err)
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
	}
}
