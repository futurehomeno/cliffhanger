package config

import (
	"github.com/futurehomeno/cliffhanger/storage"
)

// NewStorage creates a new configuration storage service.
func NewStorage[C any](cfg C, workDir string) storage.Storage[C] {
	return storage.New(cfg, workDir, Name)
}

// NewCanonicalStorage creates a new canonical configuration storage service.
func NewCanonicalStorage[C any](cfg C, workDir, defaultsDir string) storage.Storage[C] {
	return storage.NewCanonical(cfg, workDir, defaultsDir, Name)
}
