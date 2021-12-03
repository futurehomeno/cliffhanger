package routing

import (
	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/router"
)

const (
	CmdLogSetLevel = "cmd.log.set_level"
)

// RouteCmdLogSetLevel returns a routing responsible for handling the command.
func RouteCmdLogSetLevel(serviceName string, logSetter func(string) error) *router.Routing {
	return router.NewRouting(
		HandleCmdLogSetLevel(logSetter),
		router.ForService(serviceName),
		router.ForType(CmdLogSetLevel),
	)
}

// HandleCmdLogSetLevel returns a handler responsible for handling the command.
func HandleCmdLogSetLevel(logSetter func(string) error) router.MessageHandler {
	return router.NewMessageHandler(
		func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
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
		},
	)
}
