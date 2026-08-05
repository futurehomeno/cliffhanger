package router

import (
	"strings"

	log "github.com/sirupsen/logrus"
)

// DefaultLogStats returns a stats callback logging incoming and outgoing FIMP messages.
// Values of incoming messages whose interface matches one of the provided prefixes
// (e.g. "cmd.auth.") are redacted, as they may carry plaintext credentials.
func DefaultLogStats(redactedInterfacePrefixes ...string) func(Stats) {
	return func(stats Stats) {
		if stats.InputMessage != nil && stats.InputMessage.Payload != nil && log.IsLevelEnabled(log.InfoLevel) {
			payload := stats.InputMessage.Payload
			value := payload.Value

			for _, prefix := range redactedInterfacePrefixes {
				if strings.HasPrefix(payload.Interface, prefix) {
					value = "***"

					break
				}
			}

			log.Infof("FIMP %s -> %s %s %v", payload.Source, payload.Service, payload.Interface, value)
		}

		if stats.OutputMessage != nil && stats.OutputMessage.Payload != nil && log.IsLevelEnabled(log.DebugLevel) {
			log.Debugf("FIMP <- %s %s", stats.OutputMessage.Payload.Service, stats.OutputMessage.Payload.Interface)
		}
	}
}
