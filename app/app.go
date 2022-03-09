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

// ResettableApp is an interface representing app with additional functionalities.
type ResettableApp interface {
	// Reset cleans up all saved settings and introduced changes by the application.
	Reset() error
}

// InitializableApp is an interface representing app with additional functionalities.
type InitializableApp interface {
	// Initialize performs initialization of the application during its startup.
	// If error is returned application lifecycle state is changed to startup error.
	// While in startup error reinitialization is later retried within configured period.
	Initialize() error
}

// CheckableApp is an interface representing app with additional functionalities.
type CheckableApp interface {
	// Check performs periodic checks of the application status.
	// Check is performed only if application is in running state.
	Check() error
}

// Credentials is an object representing credentials for the app to log into a third-party service.
type Credentials struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	Encrypted bool   `json:"encrypted"`
}

type CredentialsError struct {
	Errors string `json:"errors"`
}

type LogginableApp interface {
	Login(*Credentials) error
}
