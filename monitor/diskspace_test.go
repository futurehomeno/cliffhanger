package monitor_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/monitor"
)

func TestDiskSpace_Start(t *testing.T) { //nolint:paralleltest
	ds := monitor.NewDiskSpace(100*time.Millisecond, 90)

	err := ds.Start()
	assert.NoError(t, err)

	time.Sleep(time.Millisecond * 150)

	assert.False(t, ds.DiskFull())

	err = ds.Stop()
	assert.NoError(t, err)

	err = ds.Stop()
	assert.Error(t, err)
}

func TestDiskSpace_DiskFull(t *testing.T) { //nolint:paralleltest
	ds := monitor.NewDiskSpace(100*time.Millisecond, 90)

	assert.False(t, ds.DiskFull())
}

func TestDiskSpace_Start_DiskFull(t *testing.T) { //nolint:paralleltest
	ds := monitor.NewDiskSpace(100*time.Millisecond, 10)

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
