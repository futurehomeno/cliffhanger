package debug

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/config"
)

var crashMarker = regexp.MustCompile(`--- Start test_app v1\.2\.3 \d{4}-\d\d-\d\dT\S+ ---\n`)

func resetCrashLog(t *testing.T) {
	t.Helper()

	savedOut := logrus.StandardLogger().Out
	crashLog.path, crashLog.app, crashLog.version, crashLog.marked = "", "", "", ""
	t.Cleanup(func() {
		logrus.SetOutput(savedOut)
		crashLog.path, crashLog.app, crashLog.version, crashLog.marked = "", "", "", ""
	})
}

func initCrashLogTest(t *testing.T, dir string) {
	t.Helper()

	cfg := &config.Default{LogLevel: "info", LogFormat: "text", LogFile: filepath.Join(dir, "app.log")}
	store := config.NewDefaultStore(func() *config.Default { return cfg }, func() error { return nil })

	require.NoError(t, InitializeLogger(store))
	t.Cleanup(func() { logManager.stopFlusher() })
}

func TestCrashLog_StartMarkerOncePerPath(t *testing.T) { //nolint:paralleltest
	for _, appFirst := range []bool{true, false} {
		dir := t.TempDir()
		resetCrashLog(t)

		if appFirst {
			SetCrashLogApp("test_app", "1.2.3")
		}

		initCrashLogTest(t, dir)

		if !appFirst {
			SetCrashLogApp("test_app", "1.2.3")
		}

		require.NoError(t, logManager.SetFile("other.log"))

		b, err := os.ReadFile(filepath.Join(dir, "crash.log")) //nolint:gosec
		require.NoError(t, err)
		assert.Regexp(t, "^"+crashMarker.String()+"$", string(b), "app set first: %v", appFirst)
	}
}

func TestCrashLog_OversizedMovedToBackup(t *testing.T) { //nolint:paralleltest
	dir := t.TempDir()
	crash := filepath.Join(dir, "crash.log")
	require.NoError(t, os.WriteFile(crash, make([]byte, maxCrashLogSize+1), 0600))
	resetCrashLog(t)

	initCrashLogTest(t, dir)

	fi, err := os.Stat(crash + ".1")
	require.NoError(t, err)
	assert.EqualValues(t, maxCrashLogSize+1, fi.Size())

	fi, err = os.Stat(crash)
	require.NoError(t, err)
	assert.Zero(t, fi.Size())
}

func TestCrashLog_GoroutinePanicLandsInCrashLog(t *testing.T) { //nolint:paralleltest
	if dir := os.Getenv("CLIFF_CRASH_LOG_DIR"); dir != "" {
		resetCrashLog(t)
		initCrashLogTest(t, dir)
		SetCrashLogApp("test_app", "1.2.3")

		go func() { panic("crash log test") }()

		select {}
	}

	dir := t.TempDir()
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestCrashLog_GoroutinePanicLandsInCrashLog$") //nolint:gosec
	cmd.Env = append(os.Environ(), "CLIFF_CRASH_LOG_DIR="+dir)
	out, err := cmd.CombinedOutput()
	require.Error(t, err, "the child must die of the panic: %s", out)

	b, err := os.ReadFile(filepath.Join(dir, "crash.log")) //nolint:gosec
	require.NoError(t, err)

	s := string(b)
	marker := crashMarker.FindStringIndex(s)
	require.NotNil(t, marker, s)

	panicAt := strings.Index(s, "panic: crash log test")
	require.Greater(t, panicAt, marker[0], s)
	assert.Contains(t, s[panicAt:], "goroutine ")
}
