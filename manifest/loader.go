package manifest

import (
	"fmt"

	"github.com/futurehomeno/cliffhanger/storage"
)

// Loader is an interface representing a manifest loader service.
type Loader interface {
	// Load loads a manifest from a configured path.
	Load() (*Manifest, error)
}

// NewLoader creates new instance of a loader service.
func NewLoader(workDir string) Loader {
	return &loader{
		workDir: workDir,
	}
}

// loader is an implementation of a loader service.
type loader struct {
	workDir string
}

// Load loads a manifest from a configured path.
func (l *loader) Load() (*Manifest, error) {
	s := storage.New(New(), l.workDir, Name)

	if err := s.Load(); err != nil {
		return nil, fmt.Errorf("manifest loader: failed to load the manifest: %w", err)
	}

	return s.Model().(*Manifest), nil
}
