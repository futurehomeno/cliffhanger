package manifest

import (
	"fmt"

	"github.com/futurehomeno/cliffhanger/storage"
)

type Loader interface {
	Load() (*Manifest, error)
}

func NewLoader(workDir string) Loader {
	return &loader{
		workDir: workDir,
	}
}

type loader struct {
	workDir string
}

func (l *loader) Load() (*Manifest, error) {
	s := storage.New(New(), l.workDir, Name)

	if err := s.Load(); err != nil {
		return nil, fmt.Errorf("manifest loader: failed to load the manifest: %w", err)
	}

	return s.Model(), nil
}
