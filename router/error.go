package router

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"
)

// Error is a failure with an error code and a plain sentence; the router answers it with the code as value and the sentence as msg.
type Error struct {
	Code fimptype.ErrorCodeT
	Msg  string
	Err  error
	// NoLog skips the router's log line, for a refusal the adapter already logged once in its period. The report is still sent.
	NoLog bool
}

// NewError creates a coded error; err is the cause, kept for the log only.
func NewError(code fimptype.ErrorCodeT, msg string, err error) *Error {
	return &Error{Code: code, Msg: msg, Err: err}
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.Msg
	}

	return e.Msg + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}

// NewErrorReport logs a failure found after the request was accepted and returns its evt.error.report, addressed and correlated to the request, for the caller to publish.
// It returns nil when the request's resp_to cannot be parsed, as the router then sends no reply either.
func NewErrorReport(request *fimpgo.Message, err error) *fimpgo.Message {
	addr := evtAddress(request.Addr)
	if request.Payload.ResponseToTopic != "" {
		respTo, parseErr := fimpgo.NewAddressFromString(request.Payload.ResponseToTopic)
		if parseErr != nil {
			log.WithField("topic", request.Topic).
				WithField("resp_to", request.Payload.ResponseToTopic).
				Errorf("[router] Parse response topic of late failure. err: %v, failure: %v", parseErr, err)

			return nil
		}

		addr = respTo
	}

	return errorReport(request, addr, err)
}

func errorReport(request *fimpgo.Message, addr *fimpgo.Address, err error) *fimpgo.Message {
	code, msg, level := fimptype.ErrorCodeInternal, err.Error(), log.ErrorLevel

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		msg = strings.Replace(msg, urlErr.Error(), urlErr.Op+": "+fmt.Sprint(urlErr.Err), 1)
	}

	var coded *Error
	if errors.As(err, &coded) && coded.Code != "" {
		code, msg = coded.Code, coded.Msg
		if code != fimptype.ErrorCodeInternal {
			level = log.WarnLevel
		}
	}

	if coded == nil || !coded.NoLog {
		log.WithField("topic", request.Topic).
			WithField("service", request.Payload.Service).
			WithField("type", request.Payload.Interface).
			WithField("code", code).
			Logf(level, "[router] Process message. err: %v", err)
	}

	reply := &fimpgo.Message{
		Addr: addr,
		Payload: fimpgo.NewMessage(
			EvtErrorReport,
			request.Payload.Service,
			fimptype.VTypeString,
			code.Str(),
			map[string]string{
				PropertyMsg:        msg,
				PropertyCmdTopic:   request.Topic,
				PropertyCmdService: request.Payload.Service.Str(),
				PropertyCmdType:    request.Payload.Interface,
			},
			nil,
			request.Payload,
		),
	}

	// Do not store device errors in the storage.
	if addr.ResourceType == fimptype.ResourceTypeDevice && reply.Payload.Storage == nil {
		reply.Payload.WithStorageStrategy(fimpgo.StorageStrategySkip, "")
	}

	return reply
}
