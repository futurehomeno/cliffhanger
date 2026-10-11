package monitor_test

import (
	"testing"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/monitor"
)

func TestDiskSpace_Start(t *testing.T) { //nolint:paralleltest
	ds := monitor.NewDiskSpace(100*time.Millisecond, 101)

	err := ds.Start()
	assert.NoError(t, err)

	time.Sleep(time.Millisecond * 150)

	assert.False(t, ds.DiskFull())

	err = ds.Stop()
	assert.NoError(t, err)

	err = ds.Stop()
	assert.Error(t, err)
}

func TestDiskSpace_DiskFull_SamplesOnDemandBeforeStart(t *testing.T) { //nolint:paralleltest
	usage, err := disk.Usage("/")
	require.NoError(t, err)

	if usage.UsedPercent < 2 || usage.UsedPercent > 98 {
		t.Skipf("host disk usage %.1f%% leaves no margin around the limit", usage.UsedPercent)
	}

	assert.True(t, monitor.NewDiskSpace(time.Hour, usage.UsedPercent-1).DiskFull())
	assert.False(t, monitor.NewDiskSpace(time.Hour, usage.UsedPercent+1).DiskFull())
}

func TestDiskSpace_DiskFull_HostIndependentLimits(t *testing.T) { //nolint:paralleltest
	assert.False(t, monitor.NewDiskSpace(time.Hour, 101).DiskFull())
	assert.True(t, monitor.NewDiskSpace(time.Hour, 0).DiskFull())
}

func TestDiskSpace_Start_DiskFull(t *testing.T) { //nolint:paralleltest
	ds := monitor.NewDiskSpace(100*time.Millisecond, 0)

	err := ds.Start()
	assert.NoError(t, err)

	time.Sleep(time.Millisecond * 150)

	assert.True(t, ds.DiskFull())

	err = ds.Stop()
	assert.NoError(t, err)
}

func TestDiskSpace_Start_SecondStartErrors(t *testing.T) { //nolint:paralleltest
	ds := monitor.NewDiskSpace(100*time.Millisecond, 50)

	err := ds.Start()
	assert.NoError(t, err)

	err = ds.Start()
	assert.Error(t, err)

	err = ds.Stop()
	assert.NoError(t, err)
}
