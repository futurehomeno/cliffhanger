package root

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/discovery"
	mockedroot "github.com/futurehomeno/cliffhanger/test/mocks/root"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

// TestWait_StopResultStaysWithItsRun covers a Wait that saw the app running and released the
// lock: a Stop and a new Start before its receive must not take its stop result, and a Wait on
// the new run must not return the previous one's.
func TestWait_StopResultStaysWithItsRun(t *testing.T) { //nolint:paralleltest
	firstStop := errors.New("first stop")

	service := mockedroot.NewService(t).MockStart(nil)
	service.On("Stop").Return(firstStop).Once()
	service.On("Stop").Return(nil).Once()

	built, err := NewCoreAppBuilder().
		WithMQTT(suite.DefaultMQTT("root_wait_restart", "", "", "")).
		WithServiceDiscovery("test_app", discovery.ResourceTypeApp, "test_app", "1", "1.0.0").
		WithServices(service).
		Build()
	require.NoError(t, err)

	a := built.(*app) //nolint:forcetypeassert

	require.NoError(t, a.Start())

	committed := a.result()

	require.ErrorIs(t, a.Stop(), firstStop)
	require.NoError(t, a.Start())

	select {
	case err := <-committed:
		assert.ErrorIs(t, err, firstStop)
	case <-time.After(time.Second):
		t.Fatal("the committed Wait never got its run's stop result")
	}

	go func() {
		time.Sleep(100 * time.Millisecond)

		_ = a.Stop()
	}()

	assert.NoError(t, a.Wait())
}
