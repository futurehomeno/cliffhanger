package suite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/google/uuid"

	"github.com/futurehomeno/cliffhanger/router"
)

// Router is an MQTT router used for testing purposes.
// It allows to set expectations for incoming messages and assert if they have been met.
type Router struct {
	t      *testing.T
	mqtt   *fimpgo.MqttTransport
	router router.Router

	mu           sync.RWMutex
	expectations []*Expectation

	registryMu      sync.RWMutex
	messageRegistry map[*Expectation]*messageBucket
}

// NewTestRouter creates new instance of a Router.
func NewTestRouter(t *testing.T, mqtt *fimpgo.MqttTransport) *Router {
	t.Helper()

	r := &Router{
		t:    t,
		mqtt: mqtt,
	}

	channelID := "test-router-" + uuid.New().String()
	r.router = router.NewRouter(mqtt, channelID, r.expectationsRouting())
	//WithOptions(
	//	router.WithAsyncProcessing(5),
	//	router.WithMessageBuffer(20),
	//)

	return r
}

// Start starts the router and initiates processing of incoming messages.
func (r *Router) Start() {
	r.t.Helper()

	r.cleanUpExpectations()

	if err := r.router.Start(); err != nil {
		r.t.Fatalf("failed to start the router: %s", err)
	}
}

// Stop stops the router.
func (r *Router) Stop() {
	r.t.Helper()

	if err := r.router.Stop(); err != nil {
		r.t.Fatalf("failed to stop the router: %s", err)
	}
}

// Expect adds expectations to the router.
func (r *Router) Expect(e ...*Expectation) {
	r.t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	r.expectations = append(r.expectations, e...)
}

// AssertExpectations checks if all expectations have been met.
// Accepts a timeout as a parameter. If the timeout is reached before all expectations are met, the test fails.
func (r *Router) AssertExpectations(timeout time.Duration) {
	r.t.Helper()

	defer r.cleanUpExpectations()

	t := time.NewTimer(timeout)
	defer t.Stop()

	waitUntilTimeout := r.shouldWaitUntilTimeout()

	for {
		select {
		case <-t.C:
			if !r.expectationsMet() {
				r.t.Errorf(r.failedExpectationsMessage())
			}

			return
		default:
			if waitUntilTimeout {
				continue
			}

			if r.expectationsMet() {
				return
			}
		}
	}
}

func (r *Router) expectationsMet() bool {
	r.t.Helper()

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, e := range r.expectations {
		if !e.assert() {
			return false
		}
	}

	return true
}

func (r *Router) shouldWaitUntilTimeout() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, e := range r.expectations {
		if e.Occurrence == Never {
			return true
		}
	}

	return false
}

func (r *Router) failedExpectationsMessage() string {
	var sb strings.Builder

	sb.WriteString("Test router: some expectations have not been met:\n")

	r.mu.RLock()
	defer r.mu.RUnlock()

	for i, e := range r.expectations {
		if e.assert() {
			continue
		}

		sb.WriteString("---------------------------------------------------------------------------\n")
		sb.WriteString(fmt.Sprintf("Expectation #%d, occurrence: %s, called times: %d\n", i, e.Occurrence, e.called))

		r.registryMu.RLock()
		item, ok := r.messageRegistry[e]
		r.registryMu.RUnlock()

		if !ok {
			continue
		}

		sb.WriteString("\nThe closest messages I have are:\n")

		for _, m := range item.messages {
			sb.WriteString(fmt.Sprintf("\nTopic: %s\n", getMessageTopic(r.t, m)))

			b, err := m.Payload.SerializeToJson()
			if err != nil {
				sb.WriteString(fmt.Sprintf("The message could not be serialized to JSON: %s\n", err))

				continue
			}

			var buf bytes.Buffer
			if err = json.Indent(&buf, b, "", "  "); err != nil {
				sb.WriteString(fmt.Sprintf("The message could not be indented: %s\n", err))

				continue
			}

			sb.Write(buf.Bytes())
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")

	return sb.String()
}

func (r *Router) cleanUpExpectations() {
	r.t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	r.expectations = nil

	r.registryMu.Lock()
	defer r.registryMu.Unlock()

	r.messageRegistry = make(map[*Expectation]*messageBucket)
}

func (r *Router) expectationsRouting() *router.Routing {
	r.t.Helper()

	return router.NewRouting(router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			return r.processMessage(message)
		}),
	))
}

func (r *Router) processMessage(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
	r.t.Helper()

	r.mu.RLock()
	expectations := r.expectations
	r.mu.RUnlock()

	for _, e := range expectations {
		voted, votes := e.vote(message)

		r.registerIncomingMessage(message, e, votes)

		if !voted {
			continue
		}

		r.mu.Lock()
		e.called++
		r.mu.Unlock()

		if e.PublishFn != nil {
			e.Publish = e.PublishFn()
		}

		publishMessage(r.t, r.mqtt, e.Publish)

		if e.ReplyFn != nil {
			e.Reply = e.ReplyFn()
		}

		return e.Reply, nil
	}

	return nil, nil
}

func (r *Router) registerIncomingMessage(message *fimpgo.Message, expectation *Expectation, votes int) {
	if votes == 0 {
		return
	}

	r.registryMu.Lock()
	defer r.registryMu.Unlock()

	bucket, ok := r.messageRegistry[expectation]
	if !ok {
		r.messageRegistry[expectation] = &messageBucket{
			votes:    votes,
			messages: []*fimpgo.Message{message},
		}

		return
	}

	if votes < bucket.votes {
		return
	}

	if votes > bucket.votes {
		bucket.votes = votes
		bucket.messages = []*fimpgo.Message{message}

		return
	}

	bucket.messages = append(bucket.messages, message)
}

func publishMessage(t *testing.T, mqtt *fimpgo.MqttTransport, message *fimpgo.Message) {
	t.Helper()

	if message == nil {
		return
	}

	topic := getMessageTopic(t, message)
	if err := mqtt.PublishToTopic(topic, message.Payload); err != nil {
		t.Fatalf("failed to publish a message: %s", err)
	}
}

func getMessageTopic(t *testing.T, message *fimpgo.Message) string {
	t.Helper()

	if message.Topic != "" {
		return message.Topic
	}

	return message.Addr.Serialize()
}

type messageBucket struct {
	votes    int
	messages []*fimpgo.Message
}
