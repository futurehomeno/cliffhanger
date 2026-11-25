package storage_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
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

func TestNewCanonicalState_ResetKeepsModel(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()

	s := storage.NewCanonicalState(&secrets{AccessToken: "keep-me"}, workDir, "state.json")
	require.NoError(t, s.Save())

	require.NoError(t, s.Reset(), "reset with no defaults must not error for a non-secret store")

	require.NotNil(t, s.Model())
	assert.Equal(t, "keep-me", s.Model().AccessToken,
		"a no-defaults state store (adapter.json, config caches) is not a secrets store; "+
			"reset must only remove the file, not wipe the in-memory model")

	_, err := os.Stat(filepath.Join(workDir, "state.json"))
	assert.True(t, os.IsNotExist(err), "reset must still remove the persisted file")
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

type typedSecrets struct {
	AccessToken string
	PIN         int
	Quoted      int `json:",string"`
	ExpiresAt   time.Time
}

func TestNewSecrets_LoadErrorOmitsFileContents(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name, body, secret, kind string
	}{
		{"cut short", `{"AccessToken": "s3cr3t"`, "s3cr3t", "not valid JSON"},
		{"number overflow", `{"PIN": 98765432109876543210}`, "98765432109876543210", "a value does not fit the model"},
		{"string tag", `{"Quoted": "FAKE-SECRET-1"}`, "FAKE-SECRET-1", "a value does not fit the model"},
		{"time field", `{"ExpiresAt": "FAKE-SECRET-2"}`, "FAKE-SECRET-2", "a value does not fit the model"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			workDir := t.TempDir()
			path := filepath.Join(workDir, "data", "secrets.json")
			require.NoError(t, os.MkdirAll(filepath.Join(workDir, "data"), 0o750))
			require.NoError(t, os.WriteFile(path, []byte(tc.body), 0o600))

			err := storage.NewSecrets(&typedSecrets{}, workDir, "secrets.json").Load()
			require.Error(t, err)
			assert.NotContains(t, err.Error(), tc.secret, "a load error ends up in logs, so it must not carry the file's contents")
			assert.Equal(t, fmt.Sprintf("storage: cannot unmarshal the secrets file at path %s (%d bytes): %s", path, len(tc.body), tc.kind), err.Error())
		})
	}
}

func TestNewSecrets_BackupFallbackLogOmitsFileContents(t *testing.T) {
	hook := logtest.NewGlobal()
	defer log.StandardLogger().ReplaceHooks(make(log.LevelHooks))

	workDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(workDir, "data"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "data", "secrets.json"), []byte(`{"ExpiresAt": "FAKE-SECRET-3"}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "data", "secrets.json.bak"), []byte(`{"AccessToken": "token"}`), 0o600))

	s := storage.NewSecrets(&typedSecrets{}, workDir, "secrets.json")
	require.NoError(t, s.Load())
	assert.Equal(t, "token", s.Model().AccessToken)

	require.NotEmpty(t, hook.AllEntries())
	assert.Contains(t, hook.LastEntry().Message, "(30 bytes): a value does not fit the model")

	for _, entry := range hook.AllEntries() {
		line, err := entry.String()
		require.NoError(t, err)
		assert.NotContains(t, line, "FAKE-SECRET-3")
	}
}
