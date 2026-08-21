package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/event"
)

func TestPublishConfigurationChange(t *testing.T) {
	t.Parallel()

	manager := event.NewManager()
	ch := manager.Subscribe("test", 5, config.WaitForConfigurationUpdate("srv", "interval"))

	defer manager.Unsubscribe("test")

	config.PublishConfigurationChange("srv", "interval", config.WithConfigurationChangeEvent(manager))

	select {
	case <-ch:
	case <-time.After(time.Second):
		assert.Fail(t, "expected a configuration change event for the subscribed setting")
	}
}
