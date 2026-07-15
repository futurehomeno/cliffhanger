package auth_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/auth"
)

func TestTokenExpirationDate(t *testing.T) {
	t.Parallel()

	expiry := time.Now().Add(time.Hour).Truncate(time.Second).UTC()

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expiry),
	}).SignedString([]byte("secret"))
	assert.NoError(t, err)

	got, err := auth.TokenExpirationDate(token)
	assert.NoError(t, err)
	assert.Equal(t, expiry, got)

	noExpiry, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{}).SignedString([]byte("secret"))
	assert.NoError(t, err)

	_, err = auth.TokenExpirationDate(noExpiry)
	assert.Error(t, err)

	_, err = auth.TokenExpirationDate("not-a-token")
	assert.Error(t, err)
}
