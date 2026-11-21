package chargepoint

import (
	"errors"
	"fmt"
	"testing"

	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter"
)

type stateReportService struct {
	Service
	err error
}

func (s stateReportService) SendStateReport(bool) (bool, error) { return false, s.err }

func TestSendStateReportAfterError(t *testing.T) { //nolint:paralleltest
	hook := logtest.NewGlobal()
	defer log.StandardLogger().ReplaceHooks(make(log.LevelHooks))

	sendStateReportAfterError(stateReportService{err: fmt.Errorf("no state: %w", adapter.ErrNotReported)})
	assert.Empty(t, hook.AllEntries(), "state not reported yet")

	sendStateReportAfterError(stateReportService{err: errors.New("state read failed")})
	assert.Equal(t, log.ErrorLevel, hook.LastEntry().Level)
}
