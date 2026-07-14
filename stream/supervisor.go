package stream

import (
	"context"
	"errors"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/backoff"
)

// Connection dials and serves a streaming connection, blocking until the connection ends
// or ctx is cancelled. It must call connected once the connection is established, resetting
// the reconnection backoff. A nil error means a clean shutdown.
type Connection func(ctx context.Context, connected func()) error

// Supervisor keeps a single streaming connection alive, reconnecting with backoff and
// implementing root.Service. Subscription registries and resubscription remain with the caller.
type Supervisor struct {
	connect Connection
	backoff backoff.Stateful

	mu      sync.Mutex
	stop    context.CancelFunc
	trigger context.CancelFunc
	done    chan struct{}
}

// NewSupervisor creates a supervisor for the connection, using a default backoff if b is nil.
func NewSupervisor(connect Connection, b backoff.Stateful) *Supervisor {
	if b == nil {
		b = backoff.NewStateful(10*time.Second, time.Minute, 5*time.Minute, 1, 3)
	}

	return &Supervisor{connect: connect, backoff: b}
}

func (s *Supervisor) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.done != nil {
		return errors.New("stream: supervisor already started")
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.stop = cancel
	s.done = make(chan struct{})

	go s.run(ctx, s.done)

	return nil
}

func (s *Supervisor) Stop() error {
	s.mu.Lock()

	if s.done == nil {
		s.mu.Unlock()

		return nil
	}

	s.stop()
	done := s.done
	s.done = nil
	s.mu.Unlock()

	<-done

	return nil
}

// TriggerReconnect drops the current connection or skips a pending backoff wait,
// forcing an immediate reconnection, e.g. after credentials change.
func (s *Supervisor) TriggerReconnect() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.done != nil && s.trigger != nil {
		s.trigger()
	}
}

func (s *Supervisor) run(ctx context.Context, done chan struct{}) {
	defer close(done)

	for {
		connCtx, cancel := context.WithCancel(ctx)
		s.setTrigger(cancel)

		err := s.connect(connCtx, s.backoff.Reset)

		if ctx.Err() != nil {
			cancel()

			return
		}

		delay := s.backoff.Next()
		if err != nil {
			log.Warnf("[stream] Connection err: %v, reconnecting in %s", err, delay)
		} else {
			log.Infof("[stream] Connection ended, reconnecting in %s", delay)
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			cancel()

			return
		case <-connCtx.Done():
			// Triggered during the connection or the wait - reconnect immediately.
			timer.Stop()
		case <-timer.C:
		}

		cancel()
	}
}

func (s *Supervisor) setTrigger(cancel context.CancelFunc) {
	s.mu.Lock()
	s.trigger = cancel
	s.mu.Unlock()
}
