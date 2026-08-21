package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtParser = jwt.NewParser()

// TokenExpirationDate returns the expiration date (UTC) of the JWT token without verifying its signature.
func TokenExpirationDate(jwtToken string) (time.Time, error) {
	var claims jwt.RegisteredClaims

	_, _, err := jwtParser.ParseUnverified(jwtToken, &claims)
	if err != nil {
		return time.Time{}, err
	}

	if claims.ExpiresAt == nil {
		return time.Time{}, errors.New("no expiration date found in the token")
	}

	return claims.ExpiresAt.UTC(), nil
}
