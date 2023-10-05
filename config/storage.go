package config

import (
	"github.com/futurehomeno/cliffhanger/bootstrap"
	"github.com/futurehomeno/cliffhanger/storage"
)

func NewDefaultStorage[T any](cfg T) storage.Storage[T] {
	return NewStorage(cfg, bootstrap.GetWorkingDirectory())
}

// NewStorage creates a new configuration storage service.
func NewStorage[T any](cfg T, workDir string) storage.Storage[T] {
	return storage.New(cfg, workDir, Name)
}

// NewCanonicalStorage creates a new canonical configuration storage service.
func NewCanonicalStorage[T any](cfg T, workDir, defaultsDir string) storage.Storage[T] {
	return storage.NewCanonical(cfg, workDir, defaultsDir, Name)
}
