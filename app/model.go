package app

import (
	"github.com/futurehomeno/cliffhanger/lifecycle"
)

const (
	OperationStatusOK    = "ok"
	OperationStatusError = "error"
)

type ConfigurationReport struct {
	OpStatus string              `json:"op_status"`
	OpError  string              `json:"op_error"`
	AppState lifecycle.AppStateT `json:"app_state"`
}

type AuthenticationReport struct {
	Status    string `json:"status"`
	ErrorText string `json:"error_text"`
	ErrorCode string `json:"error_code"`
	Errors    string `json:"errors,omitempty"` // Redundant and deprecated field to maintain compatibility with FHX.
}

type LoginCredentials struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	Encrypted bool   `json:"encrypted"`
}
