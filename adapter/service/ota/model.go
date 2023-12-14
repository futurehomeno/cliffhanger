package ota

import (
	"fmt"
)

// Status represents an OTA update status.
type Status int

const (
	StatusIdle       Status = iota // StatusIdle indicates that there's no ongoing OTA update.
	StatusStarted                  // StatusStarted indicates that an OTA update has been started, but we don't know its progress yet.
	StatusInProgress               // StatusInProgress indicates that an OTA update is in progress.
	StatusDone                     // StatusDone indicates that an OTA update has been completed.
)

func (s Status) isValid() bool {
	for _, status := range allowedStatuses() {
		if s == status {
			return true
		}
	}

	return false
}

func allowedStatuses() []Status {
	return []Status{
		StatusIdle,
		StatusStarted,
		StatusInProgress,
		StatusDone,
	}
}

// StatusReport represents an OTA update report.
type StatusReport struct {
	Status   Status
	Progress ProgressData // must be provided for StatusInProgress
	Result   ResultData   // must be provided for StatusDone
}

func (r StatusReport) validate() error {
	if !r.Status.isValid() {
		return fmt.Errorf("invalid status: %d", r.Status)
	}

	switch r.Status { //nolint:exhaustive
	case StatusInProgress:
		return r.Progress.validate()
	case StatusDone:
		return r.Result.validate()
	default:
		return nil
	}
}

// ProgressData represents 'in progress' data of an OTA update.
type ProgressData struct {
	Progress         int
	RemainingMinutes int
	RemainingSeconds int
}

func (d ProgressData) validate() error {
	if d.Progress < 0 || d.Progress > 100 {
		return fmt.Errorf("progress must be between 0 and 100")
	}

	if d.RemainingMinutes < 0 {
		return fmt.Errorf("remaining minutes must be greater than or equal 0")
	}

	if d.RemainingSeconds < 0 {
		return fmt.Errorf("remaining seconds must be greater than or equal 0")
	}

	return nil
}

// ResultData represents 'result' data of an OTA update.
type ResultData struct {
	Error Error
}

func (d ResultData) validate() error {
	if !d.Error.isValid() {
		return fmt.Errorf("invalid error: %s", d.Error)
	}

	return nil
}

// EndReport represents an OTA update end report propagated via FIMP.
type EndReport struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// Error represents an OTA update error.
type Error string

func (e Error) String() string {
	return string(e)
}

func (e Error) isValid() bool {
	for _, err := range allowedErrors() {
		if e == err {
			return true
		}
	}

	return false
}

const (
	ErrLowBattery      = "low_battery"
	ErrInvalidImage    = "invalid_image"
	ErrNotUpgradable   = "not_upgradable"
	ErrNeedsUserAction = "needs_user_action"
	ErrOther           = "other"
	ErrNoError         = ""
)

func allowedErrors() []Error {
	return []Error{
		ErrLowBattery,
		ErrInvalidImage,
		ErrNotUpgradable,
		ErrNeedsUserAction,
		ErrOther,
		ErrNoError,
	}
}
