package app_test

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/manifest"
	"github.com/futurehomeno/cliffhanger/task"
)

type stubCheckableApp struct {
	calls         int
	err           error
	checkInterval time.Duration
}

func (s *stubCheckableApp) GetManifest() (*manifest.Manifest, error) { return nil, nil }
func (s *stubCheckableApp) Configure(any) error                      { return nil }
func (s *stubCheckableApp) Uninstall() error                         { return nil }
func (s *stubCheckableApp) ErrorsReport() ([]string, error)          { return nil, nil }
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

type stubInitApp struct {
	stubCheckableApp

	calls atomic.Int32
	errs  []error
}

func (s *stubInitApp) Initialize() error {
	n := int(s.calls.Add(1))
	if n <= len(s.errs) {
		return s.errs[n-1]
	}

	return nil
}

func TestHandleInitialization(t *testing.T) {
	t.Parallel()

	t.Run("failure marks startup error only", func(t *testing.T) {
		t.Parallel()

		lc := lifecycle.New(nil)
		lc.SetConfigState(lifecycle.ConfigStateConfigured)
		lc.SetConnState(lifecycle.ConnStateConnected)
		lc.SetAuthState(lifecycle.AuthStateAuthenticated)

		require.NotPanics(t, app.HandleInitialization(&stubInitApp{errs: []error{errors.New("boom")}}, lc, time.Minute))

		assert.Equal(t, lifecycle.AppHealthStartupError, lc.AppHealth())
		assert.Equal(t, lifecycle.ConfigStateConfigured, lc.ConfigState())
		assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
		assert.Equal(t, lifecycle.AuthStateAuthenticated, lc.AuthState())
	})

	t.Run("success leaves the lifecycle alone", func(t *testing.T) {
		t.Parallel()

		lc := lifecycle.New(nil)
		before := *lc.AllStates()

		app.HandleInitialization(&stubInitApp{}, lc, time.Minute)()

		assert.Equal(t, before, *lc.AllStates())
	})
}

func TestTaskInitialization_RetriesOnlyWhileInStartupError(t *testing.T) {
	t.Parallel()

	run := func(t *testing.T, stub *stubInitApp) *lifecycle.Lifecycle {
		t.Helper()

		lc := lifecycle.New(nil)
		manager := task.NewManager(app.TaskInitialization(stub, lc, 10*time.Millisecond)...)
		require.NoError(t, manager.Start())
		t.Cleanup(func() { _ = manager.Stop() })

		return lc
	}

	t.Run("failed startup is retried", func(t *testing.T) {
		t.Parallel()

		stub := &stubInitApp{errs: []error{errors.New("boom")}}
		lc := run(t, stub)

		assert.Eventually(t, func() bool { return stub.calls.Load() >= 2 }, time.Second, 5*time.Millisecond)
		assert.Equal(t, lifecycle.AppHealthStartupError, lc.AppHealth())
	})

	t.Run("successful startup is not repeated", func(t *testing.T) {
		t.Parallel()

		stub := &stubInitApp{}
		run(t, stub)

		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, int32(1), stub.calls.Load())
	})
}

func TestTaskApp_InitializableAndCheckableApp(t *testing.T) {
	t.Parallel()

	assert.Len(t, app.TaskApp(&stubInitApp{}, lifecycle.New(nil)), 3)
}
