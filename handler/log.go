package handler

import (
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
)

// CmdLogSetLevel is a handler responsible for manipulating a log level of the application.
func CmdLogSetLevel(logSetter func(string) error) router.ErrorableHandlerFn {
	return func(message *fimpgo.Message) (*fimpgo.Message, error) {
		level, err := message.Payload.GetStringValue()
		if err != nil {
			return nil, err
		}

		logLevel, err := log.ParseLevel(level)
		if err != nil {
			return nil, err
		}

		err = logSetter(level)
		if err != nil {
			return nil, err
		}

		log.SetLevel(logLevel)
		log.Infof("Log level updated to %s", logLevel)

		return nil, nil
	}
}
