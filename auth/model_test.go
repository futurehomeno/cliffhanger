package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/auth"
	"github.com/futurehomeno/cliffhanger/hub"
)

func TestProxyURL(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "https://partners-beta.futurehome.io", auth.ProxyURL(hub.EnvBeta))
	assert.Equal(t, "https://partners.futurehome.io", auth.ProxyURL(hub.EnvProd))
	assert.Equal(t, "https://partners.futurehome.io", auth.ProxyURL("unknown"))
}

func TestProxyCallbackURL(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "https://app-static-beta.futurehome.io/playground_oauth_callback", auth.ProxyCallbackURL(hub.EnvBeta))
	assert.Equal(t, "https://app-static.futurehome.io/playground_oauth_callback", auth.ProxyCallbackURL(hub.EnvProd))
	assert.Equal(t, "https://app-static.futurehome.io/playground_oauth_callback", auth.ProxyCallbackURL("unknown"))
}
