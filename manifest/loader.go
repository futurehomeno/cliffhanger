package manifest

import (
	"fmt"

	"github.com/futurehomeno/cliffhanger/storage"
)

// Loader is an interface representing a manifest loader service.
type Loader[C any] interface {
	// Load loads a manifest from a configured path.
	Load() (*Manifest[C], error)
}

// NewLoader creates new instance of a loader service.
func NewLoader[C any](workDir string) Loader[C] {
	return &loader[C]{
		workDir: workDir,
	}
}

// loader is an implementation of a loader service.
type loader[C any] struct {
	workDir string
}

// Load loads a manifest from a configured path.
func (l *loader[C]) Load() (*Manifest[C], error) {
	s := storage.New(New[C](), l.workDir, Name)

	if err := s.Load(); err != nil {
		return nil, fmt.Errorf("manifest loader: failed to load the manifest: %w", err)
	}

	return s.Model(), nil //nolint:forcetypeassert
}
