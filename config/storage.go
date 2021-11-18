package config

import (
	"github.com/futurehomeno/cliffhanger/storage"
)

// NewStorage creates a new configuration storage service.
func NewStorage(cfg interface{}, workDir string) storage.Storage {
	return storage.New(cfg, workDir, Name)
}
