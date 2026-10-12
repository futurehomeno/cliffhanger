//go:build unix

package security_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/security"
)

// Not parallel: the umask is process-wide.
func TestGetKey_NewKeyIs0600UnderRestrictiveUmask(t *testing.T) { //nolint:paralleltest
	path := filepath.Join(t.TempDir(), "key")

	defer syscall.Umask(syscall.Umask(0o377))

	_, err := security.GetKey(path)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}
