package suite

import (
	"sync"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/router"
)

const defaultTimeout = 1500 * time.Millisecond

func NewNode(name string) *Node {
	return &Node{
		Name:    name,
		Timeout: defaultTimeout,
	}
}

type Node struct {
	Name         string
	Command      *fimpgo.Message
	Expectations []*Expectation
	Timeout      time.Duration

	lock                    *sync.RWMutex
	done                    chan struct{}
	postExpectationCallback func(t *testing.T)
}

func (n *Node) WithName(name string) *Node {
	n.Name = name

	return n
}

func (n *Node) WithCommand(command *fimpgo.Message) *Node {
	n.Command = command

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

func (n *Node) Run(t *testing.T, mqtt *fimpgo.MqttTransport) {
	t.Helper()

	n.configure()

	nodeRouter := router.NewRouter(mqtt, "router_test_node", n.prepareExpectationRouting(t, mqtt))

	err := nodeRouter.Start()
	if err != nil {
		t.Fatalf("failed to start the node router")
	}

	if n.postExpectationCallback != nil {
		n.postExpectationCallback(t)
	}

	n.publishMessage(t, mqtt, n.Command)

	select {
	case <-time.After(n.Timeout):
		break
	case <-n.done:
		break
	}

	err = nodeRouter.Stop()
	if err != nil {
		t.Fatalf("failed to stop the node router")
	}

	n.assertExpectations(t)
}

func (n *Node) configure() {
	if n.Timeout == 0 {
		n.Timeout = defaultTimeout
	}

	n.lock = &sync.RWMutex{}
	n.done = make(chan struct{})
}

func (n *Node) prepareExpectationRouting(t *testing.T, mqtt *fimpgo.MqttTransport) *router.Routing {
	t.Helper()

	return router.NewRouting(router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			return n.processMessage(t, mqtt, message)
		}),
	))
}

func (n *Node) processMessage(t *testing.T, mqtt *fimpgo.MqttTransport, message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
	t.Helper()

	defer n.checkIfDone()

	for _, e := range n.Expectations {
		if !e.vote(message) {
			continue
		}

		n.lock.Lock()

		if e.called == 1 && (e.Occurrence == ExactlyOnce || e.Occurrence == AtMostOnce) {
			n.lock.Unlock()

			continue
		}

		e.called++

		n.lock.Unlock()

		n.publishMessage(t, mqtt, e.Publish)

		return e.Reply, nil
	}

	return nil, nil
}

func (n *Node) publishMessage(t *testing.T, mqtt *fimpgo.MqttTransport, message *fimpgo.Message) {
	if message == nil {
		return
	}

	var err error

	if message.Topic != "" {
		err = mqtt.PublishToTopic(message.Topic, message.Payload)
	} else {
		err = mqtt.Publish(message.Addr, message.Payload)
	}

	if err != nil {
		t.Fatalf("failed to publish a message: %s", err)
	}
}

func (n *Node) checkIfDone() {
	n.lock.RLock()
	defer n.lock.RUnlock()

	for _, e := range n.Expectations {
		if !e.assert() {
			return
		}
	}

	close(n.done)
}

func (n *Node) assertExpectations(t *testing.T) {
	t.Helper()

	n.lock.RLock()
	defer n.lock.RUnlock()

	for i, e := range n.Expectations {
		if e.assert() {
			continue
		}

		t.Errorf("expectation #%d was not fulfilled", i)

		return
	}
}