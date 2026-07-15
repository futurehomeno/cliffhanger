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

func TestTransport_Redirects(t *testing.T) {
	t.Parallel()

	otherAuth := "unset"
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		otherAuth = r.Header.Get("Authorization")
	}))
	defer other.Close()

	sameAuth := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cross":
			http.Redirect(w, r, other.URL, http.StatusFound)
		case "/same":
			http.Redirect(w, r, "/target", http.StatusFound)
		default:
			sameAuth = r.Header.Get("Authorization")
		}
	}))
	defer srv.Close()

	client := &http.Client{Transport: &auth.Transport{Source: staticToken("token")}}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/cross", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Empty(t, otherAuth, "bearer must not leak to another host on a redirect")

	req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/same", nil)
	assert.NoError(t, err)

	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, "Bearer token", sameAuth, "bearer should survive a same host redirect")
}
