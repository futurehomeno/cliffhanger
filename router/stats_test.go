package router_test

import (
	"testing"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/router"
)

func TestDefaultLogStats(t *testing.T) {
	hook := test.NewGlobal()
	defer hook.Reset()

	log.SetLevel(log.DebugLevel)

	logStats := router.DefaultLogStats("cmd.auth.")

	logStats(router.Stats{}) // must not panic on empty stats

	logStats(router.Stats{
		InputMessage: &fimpgo.Message{Payload: &fimpgo.FimpMessage{
			Interface: "cmd.auth.login",
			Service:   "test",
			Value:     "secret-password",
		}},
	})

	logStats(router.Stats{
		InputMessage: &fimpgo.Message{Payload: &fimpgo.FimpMessage{
			Interface: "cmd.binary.set",
			Service:   "test",
			Value:     true,
		}},
		OutputMessage: &fimpgo.Message{Payload: &fimpgo.FimpMessage{
			Interface: "evt.binary.report",
			Service:   "test",
		}},
	})

	entries := hook.AllEntries()
	assert.Len(t, entries, 3)
	assert.Contains(t, entries[0].Message, "***")
	assert.NotContains(t, entries[0].Message, "secret-password")
	assert.Contains(t, entries[1].Message, "true")
	assert.Contains(t, entries[2].Message, "evt.binary.report")
}
