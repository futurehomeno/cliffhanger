package app

import (
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/debug/formatters"
)

// NewLogCapture registers a warn+ ring-buffer hook on the global logger and
// returns it as a LogProvider to embed in an app, backing cmd.app.get_diag.
func NewLogCapture() LogProvider {
	hook := formatters.NewErrorHook()
	log.AddHook(hook)

	return hook
}
