package app

import (
	"errors"
	"fmt"

	"github.com/futurehomeno/cliffhanger/lifecycle"
)

// ThingDestroyer destroys all things of an application. It is satisfied by adapter.Adapter.
type ThingDestroyer interface {
	DestroyAllThings() error
}

// Reset destroys all things, resets the configuration and marks the application as not
// configured. Optional teardown hooks run first, e.g. to cancel checks or close connections.
func Reset(appLifecycle *lifecycle.Lifecycle, things ThingDestroyer, resetConfig func() error, teardown ...func()) error {
	for _, fn := range teardown {
		fn()
	}

	// Mark not configured before the destructive steps so a partial failure still leaves the app
	// in a consistent unconfigured state, rather than "configured" over an already-wiped device
	// set that a later boot would restore. Both steps run and their errors are joined.
	appLifecycle.MarkNotConfigured()

	var errs []error

	if err := things.DestroyAllThings(); err != nil {
		errs = append(errs, fmt.Errorf("destroy things: %w", err))
	}

	if err := resetConfig(); err != nil {
		errs = append(errs, fmt.Errorf("reset configuration: %w", err))
	}

	return errors.Join(errs...)
}

// Logout clears credentials and marks the application as not configured.
// Optional teardown hooks run first, e.g. to cancel checks or close connections.
func Logout(appLifecycle *lifecycle.Lifecycle, clearCredentials func() error, teardown ...func()) error {
	for _, fn := range teardown {
		fn()
	}

	if err := clearCredentials(); err != nil {
		return fmt.Errorf("clear credentials: %w", err)
	}

	appLifecycle.MarkNotConfigured()

	return nil
}

// Authorize persists new credentials and promotes the application to running if the
// subsequent check confirms connectivity. Connectivity failures are reported through
// lifecycle states set by check itself, matching CheckableApp semantics; an error
// returned by check is a hard failure that aborts promotion and is propagated.
func Authorize(appLifecycle *lifecycle.Lifecycle, persistCredentials, check func() error) error {
	if err := persistCredentials(); err != nil {
		return fmt.Errorf("persist credentials: %w", err)
	}

	if err := check(); err != nil {
		return err
	}

	if appLifecycle.ConnectionState() == lifecycle.ConnStateConnected {
		appLifecycle.MarkRunning()
	}

	return nil
}

// ConfigModel casts the model provided to Configure to the application configuration type.
func ConfigModel[T any](model any) (T, error) {
	cfg, ok := model.(T)
	if !ok {
		return cfg, fmt.Errorf("expected configuration model of type %T, got %T", cfg, model)
	}

	return cfg, nil
}
