package handler_test

import (
	"errors"
	"github.com/futurehomeno/cliffhanger/handler"
	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewCmdLogSetLevel(t *testing.T) {
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
			msg:        logLevelFIMPMessage("string", "error"),
			wantLogLvl: log.ErrorLevel,
		},
		{
			name:      "error when checking payload value",
			logSetter: func(s string) error { return nil },
			msg:       logLevelFIMPMessage("bool", true),
			wantErr:   true,
		},
		{
			name:      "error when parsing log level",
			logSetter: func(s string) error { return nil },
			msg:       logLevelFIMPMessage("string", "dummy"),
			wantErr:   true,
		},
		{
			name:      "error when saving log level",
			logSetter: func(s string) error { return errors.New("test error") },
			msg:       logLevelFIMPMessage("string", "error"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := handler.CmdLogSetLevel(tt.logSetter)

			got, err := f(tt.msg)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantLogLvl, log.GetLevel())
		})
	}
}

func logLevelFIMPMessage(valueType string, value interface{}) *fimpgo.Message {
	return &fimpgo.Message{
		Payload: &fimpgo.FimpMessage{
			ValueType: valueType,
			Value:     value,
		},
	}
}
