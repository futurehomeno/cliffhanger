package lifecycle

// The Mark* bundles target cloud adapters with authentication. Applications keeping
// auth or connectivity at AuthStateNA/ConnStateNA should set states individually instead.

// MarkNotConfigured sets the state bundle of an unconfigured application,
// as done on uninstall, reset and logout.
func (l *Lifecycle) MarkNotConfigured() {
	l.SetAppHealth(AppHealthNotConfigured, nil)
	l.SetConfigState(ConfigStateNotConfigured)
	l.SetConnState(ConnStateDisconnected)
	l.SetAuthState(AuthStateNotAuthenticated)
}

// MarkRunning sets the state bundle of a configured, authenticated and connected application.
func (l *Lifecycle) MarkRunning() {
	l.SetAppHealth(AppHealthRunning, nil)
	l.SetConfigState(ConfigStateConfigured)
	l.SetConnState(ConnStateConnected)
	l.SetAuthState(AuthStateAuthenticated)
}
