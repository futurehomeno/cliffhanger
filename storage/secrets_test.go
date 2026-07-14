package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/storage"
)

type secrets struct {
	AccessToken string
}

func TestNewSecrets_FileMode(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()

	s := storage.NewSecrets(&secrets{AccessToken: "token"}, workDir, "secrets.json")
	assert.NoError(t, s.Save())

	info, err := os.Stat(filepath.Join(workDir, "data", "secrets.json"))
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0o640), info.Mode().Perm())

	assert.NoError(t, s.Load(), "load without data and defaults should not fail")
}

func TestNew_ConfigFileMode(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()

	s := storage.New(&secrets{}, workDir, "config.json")
	assert.NoError(t, s.Save())

	info, err := os.Stat(filepath.Join(workDir, "data", "config.json"))
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())

	assert.NoError(t, os.Chmod(filepath.Join(workDir, "data", "config.json"), 0o666)) //nolint:gosec
	assert.NoError(t, s.Save())

	info, err = os.Stat(filepath.Join(workDir, "data", "config.json"))
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm(), "save should tighten permissions of pre-existing files")
}
