package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/event"
)

func TestPublishConfigurationChanges(t *testing.T) {
	t.Parallel()

	manager := event.NewManager()
	ch := manager.Subscribe("test", 5, config.WaitForConfigurationUpdate("srv", "interval"))

	defer manager.Unsubscribe("test")

	config.PublishConfigurationChanges(manager, "srv", "mode", "interval")

	select {
	case <-ch:
	case <-time.After(time.Second):
		assert.Fail(t, "expected a configuration change event for the subscribed setting")
	}
}
