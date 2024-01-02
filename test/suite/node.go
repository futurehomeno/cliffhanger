package suite

import (
	"fmt"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
)

const defaultTimeout = 1500 * time.Millisecond

func SleepNode(duration time.Duration) *Node {
	return &Node{
		Name: fmt.Sprintf("Sleeping for %s", duration.String()),
		InitCallbacks: []Callback{
			func(t *testing.T) {
				t.Helper()

				time.Sleep(duration)
			},
		},
	}
}

func NewNode(name string) *Node {
	return &Node{
		Name:    name,
		Timeout: defaultTimeout,
	}
}

type Node struct {
	Name          string
	Command       *fimpgo.Message
	CommandFn     func(t *testing.T) *fimpgo.Message
	Expectations  []*Expectation
	Timeout       time.Duration
	InitCallbacks []Callback
	Callbacks     []Callback
}

func (n *Node) WithName(name string) *Node {
	n.Name = name

	return n
}

func (n *Node) WithCommand(command *fimpgo.Message) *Node {
	n.Command = command

	return n
}

func (n *Node) WithCommandFn(commandFn func(t *testing.T) *fimpgo.Message) *Node {
	n.CommandFn = commandFn

	return n
}

func (n *Node) WithExpectations(expectations ...*Expectation) *Node {
	n.Expectations = append(n.Expectations, expectations...)

	return n
}

func (n *Node) WithTimeout(timeout time.Duration) *Node {
	n.Timeout = timeout

	return n
}

func (n *Node) WithInitCallbacks(callbacks ...Callback) *Node {
	n.InitCallbacks = append(n.InitCallbacks, callbacks...)

	return n
}

func (n *Node) WithCallbacks(callbacks ...Callback) *Node {
	n.Callbacks = append(n.Callbacks, callbacks...)

	return n
}

func (n *Node) Run(t *testing.T, mqtt *fimpgo.MqttTransport) {
	t.Helper()

	n.configure()

	nodeRouter := NewTestRouter(t, mqtt)

	nodeRouter.Start()
	nodeRouter.Expect(n.Expectations...)

	for _, callback := range n.InitCallbacks {
		callback(t)
	}

	if n.CommandFn != nil {
		n.Command = n.CommandFn(t)
	}

	publishMessage(t, mqtt, n.Command)

	for _, callback := range n.Callbacks {
		callback(t)
	}

	nodeRouter.AssertExpectations(n.Timeout)
	nodeRouter.Stop()
}

func (n *Node) configure() {
	if n.Timeout == 0 {
		n.Timeout = defaultTimeout
	}
}
