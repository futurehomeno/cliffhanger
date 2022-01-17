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
	// Uninstall performs all required clean ups before uninstalling the applications.
	Uninstall() error
	// Reset resets state of the application, returning default values of configuration.
	Reset() error
}
