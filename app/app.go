package app

import (
	"time"

	"github.com/futurehomeno/cliffhanger/auth"
	"github.com/futurehomeno/cliffhanger/manifest"
)

type App interface {
	GetManifest() (*manifest.Manifest, error)
	Configure(config any) error
	Uninstall() error
}

type ResettableApp interface {
	Reset() error
}

type InitializableApp interface {
	// Initialize performs initialization of the application during its startup.
	// If error is returned application lifecycle state is changed to startup error.
	// While in startup error reinitialization is later retried within configured period.
	Initialize() error
}

type CheckableApp interface {
	// Check performs periodic checks of the application status.
	// Check is performed only if application is in running state.
	Check() error
	// CheckInterval returns the interval between Check calls.
	// Return 0 to use DefaultCheckInterval.
	CheckInterval() time.Duration
}

type LogginableApp interface {
	LogoutableApp

	// Login performs login of the application into a third party app and persistence of credentials in local storage.
	Login(credentials *LoginCredentials) error
}

type AuthorizableApp interface {
	LogoutableApp

	// Authorize performs authorization of the application into a third party app and persistence of credentials in local storage.
	Authorize(credentials *auth.OAuth2TokenResponse) error
}

type LogoutableApp interface {
	// Logout performs logout of the application from a third party app and removal of credentials in local storage.
	Logout() error
}
