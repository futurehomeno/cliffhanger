package app

import (
	"github.com/futurehomeno/cliffhanger/manifest"
)

// App is an interface representing a service responsible for preparing an application manifest and configuring app.
type App interface {
	// GetManifest returns the manifest object based on current application state and configuration.
	GetManifest() (*manifest.Manifest, error)
	// Configure performs update of the application state based on the provided configuration.
	Configure(config interface{}) error
	// Uninstall performs all required clean ups before uninstalling the application.
	Uninstall() error
}

// ExtendedApp is an interface representing App with additional functionalities.
type ExtendedApp interface {
	App
	// Reset cleans up all saved settings and introduced changes by the application.
	Reset() error
}
