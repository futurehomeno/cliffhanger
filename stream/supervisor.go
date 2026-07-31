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
	s.mu.Unlock()

	<-done

	// Cleared only after the goroutine exited, so a concurrent Start cannot overlap connections.
	s.mu.Lock()
	s.done = nil
	s.mu.Unlock()

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

		// A triggered drop is not a failure: reconnect immediately with a fresh backoff, so an
		// attempt failing right after e.g. a credentials change starts at the initial delay.
		if connCtx.Err() != nil {
			s.backoff.Reset()
			cancel()

			log.Infof("[stream] Reconnect triggered, reconnecting immediately")

			continue
		}

		delay := s.backoff.Next()
		if err != nil {
			log.Warnf("[stream] Connection err: %v, reconnecting in %s", err, delay)
		} else {
			// A clean ending is paced like a first retry but must not advance the streak.
			s.backoff.Reset()
			log.Infof("[stream] Connection ended, reconnecting in %s", delay)
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			cancel()

			return
		case <-connCtx.Done():
			// Triggered during the wait - reconnect immediately with a fresh backoff.
			timer.Stop()
			s.backoff.Reset()
		case <-timer.C:
		}

		cancel()

		// connCtx.Done may have won the select over its cancelled parent - do not dial again.
		if ctx.Err() != nil {
			return
		}
	}
}

func (s *Supervisor) setTrigger(cancel context.CancelFunc) {
	s.mu.Lock()
	s.trigger = cancel
	s.mu.Unlock()
}
