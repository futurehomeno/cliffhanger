package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/auth"
)

type staticToken string

func (t staticToken) AccessToken() (string, error) { return string(t), nil }

func TestTransport(t *testing.T) {
	t.Parallel()

	gotAuth := ""
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(status)
	}))
	defer srv.Close()

	unauthorized := 0
	client := &http.Client{Transport: &auth.Transport{
		Source:         staticToken("token"),
		OnUnauthorized: func() { unauthorized++ },
	}}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, "Bearer token", gotAuth)
	assert.Zero(t, unauthorized)

	status = http.StatusUnauthorized

	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, 1, unauthorized)
}
