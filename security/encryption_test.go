package security_test

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/security"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	t.Parallel()

	key, err := security.GenerateKey(32)
	require.NoError(t, err)

	encrypted, err := security.Encrypt("secret", key)
	require.NoError(t, err)

	decrypted, err := security.Decrypt(encrypted, key)
	require.NoError(t, err)
	assert.Equal(t, "secret", decrypted)
}

// TestDecrypt_ShortCiphertext pins that a stored secret too short to hold a nonce is reported as an
// error. Slicing the nonce out of it unchecked panicked, taking the adapter process down over a
// truncated value that only decryption could have noticed.
func TestDecrypt_ShortCiphertext(t *testing.T) {
	t.Parallel()

	key, err := security.GenerateKey(32)
	require.NoError(t, err)

	for _, encrypted := range []string{"", "ab", "abcd", "0123456789abcdef0123"} {
		_, err := security.Decrypt(encrypted, key)
		assert.Error(t, err, "ciphertext %q is shorter than the nonce", encrypted)
	}

	// Exactly the nonce size: passes the length check and fails authentication instead.
	_, err = security.Decrypt("0123456789abcdef01234567", key)
	assert.Error(t, err)
}

func TestDecrypt_InvalidInput(t *testing.T) {
	t.Parallel()

	key, err := security.GenerateKey(32)
	require.NoError(t, err)

	_, err = security.Decrypt("not hex", key)
	assert.Error(t, err)

	encrypted, err := security.Encrypt("secret", key)
	require.NoError(t, err)

	otherKey, err := security.GenerateKey(32)
	require.NoError(t, err)

	_, err = security.Decrypt(encrypted, otherKey)
	assert.Error(t, err, "a ciphertext must not decrypt under a different key")
}

func TestGetKey_GeneratesAndPersistsKey(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "key")

	key, err := security.GetKey(path)
	require.NoError(t, err)

	raw, err := hex.DecodeString(key)
	require.NoError(t, err)
	assert.Len(t, raw, 32)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	again, err := security.GetKey(path)
	require.NoError(t, err)
	assert.Equal(t, key, again, "a persisted key must be reused, not regenerated")

	encrypted, err := security.Encrypt("secret", key)
	require.NoError(t, err)

	decrypted, err := security.Decrypt(encrypted, again)
	require.NoError(t, err)
	assert.Equal(t, "secret", decrypted)
}

func TestGetKey_TightensPermissionsOfExistingKey(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "key")
	require.NoError(t, os.WriteFile(path, []byte("abcd"), 0o644)) //nolint:gosec
	require.NoError(t, os.Chmod(path, 0o644))                     //nolint:gosec

	key, err := security.GetKey(path)
	require.NoError(t, err)
	assert.Equal(t, "abcd", key)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestGetKey_UnwritablePath(t *testing.T) {
	t.Parallel()

	_, err := security.GetKey(filepath.Join(t.TempDir(), "missing", "key"))
	assert.Error(t, err)
}

func TestEncrypt_InvalidKey(t *testing.T) {
	t.Parallel()

	_, err := security.Encrypt("secret", "not hex")
	assert.Error(t, err)

	shortKey, err := security.GenerateKey(10)
	require.NoError(t, err)

	_, err = security.Encrypt("secret", shortKey)
	assert.Error(t, err)

	_, err = security.Decrypt("0123456789abcdef0123456789abcdef", shortKey)
	assert.Error(t, err)

	for _, size := range []int{16, 24} {
		key, err := security.GenerateKey(size)
		require.NoError(t, err)

		encrypted, err := security.Encrypt("secret", key)
		require.NoError(t, err)

		decrypted, err := security.Decrypt(encrypted, key)
		require.NoError(t, err)
		assert.Equal(t, "secret", decrypted)
	}
}
