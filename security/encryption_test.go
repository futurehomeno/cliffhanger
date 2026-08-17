package security_test

import (
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
