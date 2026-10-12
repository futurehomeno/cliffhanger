package monitor

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDiskSpace_FailedSampleKeepsPreviousReading(t *testing.T) {
	t.Parallel()

	d := NewDiskSpace(time.Hour, 90).(*diskSpace) //nolint:forcetypeassert
	d.usage = func() (float64, error) { return 95, nil }

	assert.True(t, d.DiskFull())

	d.usage = func() (float64, error) { return 0, errors.New("boom") }
	d.checkSpace()

	assert.True(t, d.DiskFull())
}

func TestDiskSpace_ResamplesWhileUsageIsZero(t *testing.T) {
	t.Parallel()

	samples := []float64{0, 60}
	d := NewDiskSpace(time.Hour, 50).(*diskSpace) //nolint:forcetypeassert
	d.usage = func() (float64, error) {
		v := samples[0]
		samples = samples[1:]

		return v, nil
	}

	assert.False(t, d.DiskFull())
	assert.True(t, d.DiskFull())
	assert.Empty(t, samples)
}
