package manifest

// Manager is an interface representing a service responsible for preparing an application manifest and configuring app.
type Manager interface {
	// Get returns the manifest object based on current application state and configuration.
	Get() (*Manifest, error)
	// Configure performs update of the application state based on the provided configuration.
	Configure(config interface{}) error
	// Uninstall performs all required clean ups before uninstalling the applications.
	Uninstall() error
}
