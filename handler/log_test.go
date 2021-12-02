package handler_test

import (
	"errors"
	"testing"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/handler"
)

func TestNewCmdLogSetLevel(t *testing.T) {
	t.Parallel()

	makeCommand := func(valueType string, value interface{}) *fimpgo.Message {
		return &fimpgo.Message{
			Payload: &fimpgo.FimpMessage{
				ValueType: valueType,
				Value:     value,
			},
			Addr: &fimpgo.Address{},
		}
	}

	tests := []struct {
		name       string
		logSetter  func(string) error
		msg        *fimpgo.Message
		want       *fimpgo.Message
		wantErr    bool
		wantLogLvl log.Level
	}{
		{
			name:       "happy path",
			logSetter:  func(s string) error { return nil },
			msg:        makeCommand("string", "error"),
			wantLogLvl: log.ErrorLevel,
		},
		{
			name:      "error when checking payload value",
			logSetter: func(s string) error { return nil },
			msg:       makeCommand("bool", true),
			wantErr:   true,
		},
		{
			name:      "error when parsing log level",
			logSetter: func(s string) error { return nil },
			msg:       makeCommand("string", "dummy"),
			wantErr:   true,
		},
		{
			name:      "error when saving log level",
			logSetter: func(s string) error { return errors.New("test error") },
			msg:       makeCommand("string", "error"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := handler.CmdLogSetLevel(tt.logSetter)

			got := f.Handle(tt.msg)

			if tt.wantErr {
				assert.NotNil(t, got)
			} else {
				assert.Nil(t, got)
				assert.Equal(t, tt.wantLogLvl, log.GetLevel())
			}
		})
	}
}
