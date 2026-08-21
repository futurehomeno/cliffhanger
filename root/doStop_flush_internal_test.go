package root

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/debug"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

// TestDoStop_FlushesLogsEvenOnFailure pins that a doStop() failure path still flushes the buffered
// log lines leading up to it - exactly where they matter most for diagnosing the failure.
// taskManager.Stop() on a never-started manager is the cheapest way to make a step of doStop()
// fail.
func TestDoStop_FlushesLogsEvenOnFailure(t *testing.T) { //nolint:paralleltest
	dir := t.TempDir()
	logFile := filepath.Join(dir, "app.log")

	cfg := &config.Default{LogLevel: "info", LogFormat: "text", LogFile: logFile}
	store := config.NewDefaultStore(
		func() *config.Default { return cfg },
		func() error { return nil },
	)

	savedLevel := logrus.GetLevel()
	savedOut := logrus.StandardLogger().Out
	t.Cleanup(func() {
		logrus.SetLevel(savedLevel)
		logrus.SetOutput(savedOut)
	})

	require.NoError(t, debug.InitializeLogger(store))

	a := &app{
		running:       true,
		mqtt:          suite.DefaultMQTT("root_doStop_flush", "", "", ""),
		messageRouter: noopRouter{},
		taskManager:   task.NewManager(), // never started: Stop() fails immediately
	}

	logrus.Info("buffered-line-before-failed-stop")

	b, err := os.ReadFile(logFile) //nolint:gosec
	require.NoError(t, err)
	require.NotContains(t, string(b), "buffered-line-before-failed-stop", "should still be buffered, not yet flushed")

	err = a.doStop()
	require.Error(t, err, "an unstarted task manager must fail Stop()")

	b, err = os.ReadFile(logFile) //nolint:gosec
	require.NoError(t, err)
	assert.Contains(t, string(b), "buffered-line-before-failed-stop",
		"doStop() must flush buffered logs even when it returns early on error")
}
