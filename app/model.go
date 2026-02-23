package app

import (
	"github.com/futurehomeno/cliffhanger/lifecycle"
)

// Constants defining configuration operation status.
const (
	OperationStatusOK    = "ok"
	OperationStatusError = "error"
)

// ConfigurationReport is an object holding information about configuration operation status.
type ConfigurationReport struct {
	OpStatus string              `json:"op_status"`
	OpError  string              `json:"op_error"`
	AppState lifecycle.AppStates `json:"app_state"`
}

// AuthenticationReport is an object holding information about authentication operation status.
type AuthenticationReport struct {
	Status    string `json:"status"`
	ErrorText string `json:"error_text"`
	ErrorCode string `json:"error_code"`
	Errors    string `json:"errors,omitempty"` // Redundant and deprecated field to maintain compatibility with FHX.
}

// LoginCredentials is an object representing credentials for the app to log into a third-party service.
type LoginCredentials struct {
	Username  string `json:"username"`
	Password  string `json:"password"` //nolint:gosec
	Encrypted bool   `json:"encrypted"`
}
