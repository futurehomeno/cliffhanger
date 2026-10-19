package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/manifest"
)

func writeManifest(t *testing.T, workDir, dir, content string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Join(workDir, dir), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, dir, manifest.Name), []byte(content), 0o600))
}

func TestLoader_Load(t *testing.T) {
	t.Parallel()

	t.Run("defaults only", func(t *testing.T) {
		t.Parallel()

		workDir := t.TempDir()
		writeManifest(t, workDir, "defaults", `{"init_flow":["connect"],"ui_blocks":[{"id":"b1"}]}`)

		m, err := manifest.NewLoader(workDir).Load()
		require.NoError(t, err)

		assert.Equal(t, []string{"connect"}, m.InitFlow)
		assert.NotNil(t, m.GetUIBlock("b1"))
	})

	t.Run("data overlays defaults", func(t *testing.T) {
		t.Parallel()

		workDir := t.TempDir()
		writeManifest(t, workDir, "defaults", `{"init_flow":["connect"],"ui_blocks":[{"id":"b1"}]}`)
		writeManifest(t, workDir, "data", `{"init_flow":["auth","connect"]}`)

		m, err := manifest.NewLoader(workDir).Load()
		require.NoError(t, err)

		assert.Equal(t, []string{"auth", "connect"}, m.InitFlow)
		assert.NotNil(t, m.GetUIBlock("b1"))
	})

	t.Run("no manifest files", func(t *testing.T) {
		t.Parallel()

		m, err := manifest.NewLoader(t.TempDir()).Load()

		require.ErrorContains(t, err, "manifest loader: failed to load the manifest")
		assert.Nil(t, m)
	})
}

func TestLoader_LoadReturnsAFreshManifest(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	writeManifest(t, workDir, "defaults", `{"ui_blocks":[{"id":"b1"}]}`)

	loader := manifest.NewLoader(workDir)

	first, err := loader.Load()
	require.NoError(t, err)

	first.GetUIBlock("b1").Hide()
	first.AppState.Health = "RUNNING"

	second, err := loader.Load()
	require.NoError(t, err)

	assert.False(t, second.GetUIBlock("b1").Hidden)
	assert.Empty(t, second.AppState.Health)
}
