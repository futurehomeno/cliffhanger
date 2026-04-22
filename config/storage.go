package config

import (
	"github.com/futurehomeno/cliffhanger/storage"
)

func NewStorage[C any](cfg C, workDir string) storage.Storage[C] {
	return storage.New(cfg, workDir, configFileName)
}

func NewCanonicalStorage[C any](cfg C, workDir, defaultsDir string) storage.Storage[C] {
	return storage.NewCanonical(cfg, workDir, defaultsDir, configFileName)
}
