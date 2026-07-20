package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestNewSecrets_ResetZeroesModel(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()

	s := storage.NewSecrets(&secrets{AccessToken: "token"}, workDir, "secrets.json")
	assert.NoError(t, s.Save())

	assert.NoError(t, s.Reset())

	require.NotNil(t, s.Model(), "reset keeps a usable (non-nil) model so Load/re-persist still work")
	assert.Empty(t, s.Model().AccessToken, "reset must clear the stored credentials in memory")

	_, err := os.Stat(filepath.Join(workDir, "data", "secrets.json"))
	assert.True(t, os.IsNotExist(err), "reset must remove the persisted secrets file")
}

func TestNewSecrets_LoadAfterResetSucceeds(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()

	s := storage.NewSecrets(&secrets{AccessToken: "token"}, workDir, "secrets.json")
	require.NoError(t, s.Save())
	require.NoError(t, s.Reset())

	// After reset the model must remain a valid unmarshal/mutation target — nilling it
	// regressed re-login into a "json: Unmarshal(nil)" error (and a nil deref on SetCredentials).
	s.Model().AccessToken = "fresh"
	require.NoError(t, s.Save())
	require.NoError(t, s.Load())
	assert.Equal(t, "fresh", s.Model().AccessToken)
}

func TestNewSecrets_ResetRemovesBackup(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()

	s := storage.NewSecrets(&secrets{AccessToken: "token"}, workDir, "secrets.json")
	assert.NoError(t, s.Save())
	assert.NoError(t, s.Save(), "second save creates the .bak backup of the first")

	backupPath := filepath.Join(workDir, "data", "secrets.json.bak")
	_, err := os.Stat(backupPath)
	assert.NoError(t, err, "second save should have produced a backup file")

	assert.NoError(t, s.Reset())

	_, err = os.Stat(backupPath)
	assert.True(t, os.IsNotExist(err), "reset must remove the secrets backup file")
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
