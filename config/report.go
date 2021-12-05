package config

import (
	"github.com/futurehomeno/cliffhanger/lifecycle"
)

// Constants defining configuration operation status.
const (
	OperationStatusOK    = "ok"
	OperationStatusError = "error"
)

// Report is an object holding information of configuration operation status.
type Report struct {
	OpStatus string              `json:"op_status"`
	AppState lifecycle.AppStates `json:"app_state"`
}
