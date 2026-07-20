package app

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/debug/formatters"
)

var (
	logCaptureMu   sync.Mutex
	logCaptureHook *formatters.ErrorHook
)

// NewLogCapture returns the LogProvider backing cmd.app.get_diag: a warn+
// ring-buffer hook on the global logger, exposed via ErrorsReport. The hook is
// installed once and shared by later calls, so repeated app construction (or
// many app.New calls across a test binary) does not stack duplicate hooks.
func NewLogCapture() LogProvider {
	logCaptureMu.Lock()
	defer logCaptureMu.Unlock()

	if logCaptureHook == nil {
		logCaptureHook = formatters.NewErrorHook()
		log.AddHook(logCaptureHook)
	}

	return logCaptureHook
}
