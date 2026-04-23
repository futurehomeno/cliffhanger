package app_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/manifest"
)

type stubCheckableApp struct {
	calls         int
	err           error
	checkInterval time.Duration
}

func (s *stubCheckableApp) GetManifest() (*manifest.Manifest, error) { return nil, nil }
func (s *stubCheckableApp) Configure(any) error                      { return nil }
func (s *stubCheckableApp) Uninstall() error                         { return nil }
func (s *stubCheckableApp) CheckInterval() time.Duration             { return s.checkInterval }
func (s *stubCheckableApp) Check() error {
	s.calls++

	return s.err
}

func TestHandleCheck_CallsCheck(t *testing.T) {
	t.Parallel()

	stub := &stubCheckableApp{}
	handler := app.HandleCheck(stub)

	handler()

	assert.Equal(t, 1, stub.calls)
}

func TestHandleCheck_LogsErrorAndContinues(t *testing.T) {
	t.Parallel()

	stub := &stubCheckableApp{err: errors.New("boom")}
	handler := app.HandleCheck(stub)

	require.NotPanics(t, handler)
	assert.Equal(t, 1, stub.calls)
}

func TestTaskApp_CheckableApp_CreatesCheckTask(t *testing.T) {
	t.Parallel()

	stub := &stubCheckableApp{checkInterval: 30 * time.Minute}
	lc := lifecycle.New(nil)

	tasks := app.TaskApp(stub, lc)

	assert.Len(t, tasks, 1)
}

func TestTaskApp_CheckableApp_RespectsCustomInterval(t *testing.T) {
	t.Parallel()

	stub := &stubCheckableApp{checkInterval: 10 * time.Minute}
	lc := lifecycle.New(nil)

	tasks := app.TaskApp(stub, lc)

	assert.Len(t, tasks, 1)
}

func TestTaskApp_CheckableApp_UsesDefaultIntervalWhenZero(t *testing.T) {
	t.Parallel()

	stub := &stubCheckableApp{checkInterval: 0}
	lc := lifecycle.New(nil)

	tasks := app.TaskApp(stub, lc)

	assert.Len(t, tasks, 1)
}
