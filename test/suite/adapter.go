package suite

import (
	"testing"

	"github.com/futurehomeno/nexus/seq"
)

var _ Service = (*SequenceAdapter)(nil)

// SequenceAdapter is an adapter of the sequence from github.com/futurehomeno/nexus/seq package that satisfies the Service interface.
type SequenceAdapter struct {
	t *testing.T

	controller *seq.Controller
	runner     *seq.Sequence
}

// NewSequenceAdapter creates a new SequenceAdapter.
func NewSequenceAdapter(t *testing.T, builder *seq.Builder) *SequenceAdapter {
	a := &SequenceAdapter{
		t:          t,
		controller: seq.NewController(),
	}

	a.runner = builder.WithController(a.controller).Build()

	return a
}

func (s *SequenceAdapter) Start() error {
	go func() {
		if err := s.runner.Run(); err != nil {
			s.t.Fatalf("failed to run sequence: %v", err)
		}
	}()

	err := <-s.controller.WaitUntilRunning()

	return err
}

func (s *SequenceAdapter) Stop() error {
	s.controller.Stop()

	return nil
}
