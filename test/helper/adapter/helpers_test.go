package adapterhelper_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
)

func TestTearDownAdapter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "data", "nested"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data", "adapter.json"), []byte("{}"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.db"), nil, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "keep.json"), nil, 0o600))

	callbacks := adapterhelper.TearDownAdapter(dir)
	require.Len(t, callbacks, 1)

	callbacks[0](t)

	assert.NoDirExists(t, filepath.Join(dir, "data"))
	assert.NoFileExists(t, filepath.Join(dir, "data.db"))
	assert.FileExists(t, filepath.Join(dir, "keep.json"))
}
