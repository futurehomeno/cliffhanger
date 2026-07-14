package lifecycle

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

// MarkAuthLost sets the state bundle of an application that lost authorization to its
// third party API, which for cloud adapters also means connectivity is lost.
func (l *Lifecycle) MarkAuthLost() {
	l.SetAuthState(AuthStateLost)
	l.SetConnState(ConnStateDisconnected)
}
