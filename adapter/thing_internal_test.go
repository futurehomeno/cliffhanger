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

func TestThing_ConnectorIOOutsideLock(t *testing.T) {
	t.Parallel()

	calls := map[string]func(Thing){
		"connectivity report": func(th Thing) { th.ConnectivityReport() },
		"send connectivity":   func(th Thing) { _, _ = th.SendConnectivityReport(false) },
		"ping":                func(th Thing) { _ = th.SendPingReport() },
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			c := &blockingConnector{entered: make(chan struct{}), release: make(chan struct{})}
			th := NewThing(stubPublisher{}, nil, &ThingConfig{
				InclusionReport: &fimptype.ThingInclusionReport{Address: "1"},
				Connector:       c,
			})

			done := make(chan struct{})

			go func() {
				call(th)
				close(done)
			}()

			<-c.entered

			updated := make(chan struct{})

			go func() {
				_ = th.Update()
				close(updated)
			}()

			select {
			case <-updated:
			case <-time.After(time.Second):
				t.Error("Update blocked while the connector was busy")
			}

			close(c.release)
			<-done
			<-updated
		})
	}
}
