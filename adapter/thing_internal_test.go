package adapter

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo/fimptype"
)

// blockingConnector holds Connectivity and Ping until released, like a connector waiting on a
// slow device or cloud.
type blockingConnector struct {
	entered chan struct{}
	release chan struct{}
}

func (c *blockingConnector) Connectivity() *ConnectivityDetails {
	c.entered <- struct{}{}
	<-c.release

	return &ConnectivityDetails{ConnStatus: ConnStatusUp}
}

func (c *blockingConnector) Ping() *PingDetails {
	c.entered <- struct{}{}
	<-c.release

	return &PingDetails{Status: PingResultSuccess}
}

var connectorCalls = map[string]func(Thing){
	"connectivity report": func(th Thing) { th.ConnectivityReport() },
	"send connectivity":   func(th Thing) { _, _ = th.SendConnectivityReport(false) },
	"ping":                func(th Thing) { _ = th.SendPingReport() },
}

func newBlockedThing(t *testing.T) (Thing, *blockingConnector) {
	t.Helper()

	c := &blockingConnector{entered: make(chan struct{}), release: make(chan struct{})}
	th := NewThing(stubPublisher{}, nil, &ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{Address: "1"},
		Connector:       c,
	})

	return th, c
}

func runAsync(f func()) chan struct{} {
	done := make(chan struct{})

	go func() {
		f()
		close(done)
	}()

	return done
}

func waitFor(t *testing.T, ch chan struct{}, what string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func TestThing_ConnectorIOOutsideLock(t *testing.T) {
	t.Parallel()

	for name, call := range connectorCalls {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			th, c := newBlockedThing(t)
			done := runAsync(func() { call(th) })

			waitFor(t, c.entered, "the connector call")

			updated := runAsync(func() { _ = th.Update() })

			select {
			case <-updated:
			case <-time.After(time.Second):
				t.Error("Update blocked while the connector was busy")
			}

			close(c.release)
			waitFor(t, done, "the connector call to return")
			waitFor(t, updated, "Update")
		})
	}
}

// A report published after a fresher one would leave the hub with a stale status, and connectors
// written before connector calls left the thing lock rely on not being called concurrently.
func TestThing_ConnectorCallsSerialized(t *testing.T) {
	t.Parallel()

	for name, call := range connectorCalls {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			th, c := newBlockedThing(t)
			first := runAsync(func() { _, _ = th.SendConnectivityReport(false) })

			waitFor(t, c.entered, "the first connector call")

			second := runAsync(func() { call(th) })

			select {
			case <-c.entered:
				t.Error("connector called while a connectivity report was in flight")
			case <-time.After(100 * time.Millisecond):
			}

			close(c.release)
			waitFor(t, first, "the first report")

			select {
			case <-c.entered:
			case <-second:
			}

			waitFor(t, second, "the second call")
		})
	}
}
