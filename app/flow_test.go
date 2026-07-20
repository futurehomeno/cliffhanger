package app_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/lifecycle"
)

type fakeDestroyer struct {
	err    error
	called bool
}

func (d *fakeDestroyer) DestroyAllThings() error {
	d.called = true

	return d.err
}

func TestReset(t *testing.T) {
	t.Parallel()

	lc := lifecycle.New(nil)
	lc.MarkRunning()

	destroyer := &fakeDestroyer{}
	tornDown := false
	resetCalled := false

	err := app.Reset(lc, destroyer, func() error { resetCalled = true; return nil }, func() { tornDown = true })

	assert.NoError(t, err)
	assert.True(t, tornDown)
	assert.True(t, destroyer.called)
	assert.True(t, resetCalled)
	assert.Equal(t, lifecycle.AppHealthNotConfigured, lc.AppHealth())

	lc.MarkRunning()

	destroyErr := &fakeDestroyer{err: errors.New("destroy err")}
	resetRan := false
	err = app.Reset(lc, destroyErr, func() error { resetRan = true; return nil })

	assert.Error(t, err)
	assert.Equal(t, lifecycle.AppHealthNotConfigured, lc.AppHealth(),
		"a partial failure still leaves the app in a consistent unconfigured state")
	assert.True(t, destroyErr.called)
	assert.True(t, resetRan, "config reset still runs even if destroy failed; the errors are joined")
}

func TestLogout(t *testing.T) {
	t.Parallel()

	lc := lifecycle.New(nil)
	lc.MarkRunning()

	err := app.Logout(lc, func() error { return nil })

	assert.NoError(t, err)
	assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())

	lc.MarkRunning()

	err = app.Logout(lc, func() error { return errors.New("clear err") })

	assert.Error(t, err)
	assert.Equal(t, lifecycle.AuthStateAuthenticated, lc.AuthState())
}

func TestAuthorize(t *testing.T) {
	t.Parallel()

	lc := lifecycle.New(nil)

	err := app.Authorize(lc, func() error { return errors.New("persist err") }, func() error { return nil })
	assert.Error(t, err)

	err = app.Authorize(lc, func() error { return nil }, func() error {
		lc.SetConnState(lifecycle.ConnStateDisconnected)

		return nil
	})
	assert.NoError(t, err)
	assert.NotEqual(t, lifecycle.AppHealthRunning, lc.AppHealth(), "should not promote when check left the app disconnected")

	err = app.Authorize(lc, func() error { return nil }, func() error { return errors.New("check err") })
	assert.Error(t, err, "hard check failure should be propagated")
	assert.NotEqual(t, lifecycle.AppHealthRunning, lc.AppHealth())

	err = app.Authorize(lc, func() error { return nil }, func() error {
		lc.SetConnState(lifecycle.ConnStateConnected)

		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, lifecycle.AppHealthRunning, lc.AppHealth())
	assert.Equal(t, lifecycle.ConfigStateConfigured, lc.ConfigState())
}

func TestConfigModel(t *testing.T) {
	t.Parallel()

	type cfg struct{ Name string }

	got, err := app.ConfigModel[*cfg](any(&cfg{Name: "test"}))
	assert.NoError(t, err)
	assert.Equal(t, "test", got.Name)

	_, err = app.ConfigModel[*cfg](any("wrong type"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected configuration model")
}
