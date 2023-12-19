package virtualmeter_test

import (
	"github.com/futurehomeno/cliffhanger/adapter/service/virtualmeter"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestStorage_Device(t *testing.T) {
	cases := []struct {
		name         string
		device       virtualmeter.Device
		setAddr      string
		expectToFind bool
	}{
		{
			name: "should save and find a device",
			device: virtualmeter.Device{
				Modes: map[string]float64{
					"test": 123,
					"beet": 321,
				},
				CurrentMode:       "mode",
				AccumulatedEnergy: 4321.32123,
				LastTimeUpdated:   time.Now().Format(time.RFC3339),
				Unit:              "W",
				Active:            true,
			},
			setAddr:      "test",
			expectToFind: true,
		},
		{
			name: "should not find a device when setting by a wrong key",
			device: virtualmeter.Device{
				Modes: map[string]float64{
					"test": 123,
					"beet": 321,
				},
			},
			setAddr:      "invalid",
			expectToFind: false,
		},
	}

	for _, vv := range cases {
		v := vv
		t.Run(v.name, func(t *testing.T) {
			storage := virtualmeter.NewStorage(workdir)
			defer adapterhelper.TearDownAdapter(workdir)[0](t)

			err := storage.SetDevice(v.setAddr, v.device)
			assert.NoError(t, err, "should set a device")

			newDev, err := storage.Device("test")

			if v.expectToFind {
				assert.Equal(t, v.device, newDev, "should find the same device as was saved")
				assert.NoError(t, err, "should not get an error when finding a device")
			} else {
				assert.Error(t, err, "should get an error")
			}

			if v.expectToFind {
				err := storage.DeleteDevice(v.setAddr)
				assert.NoError(t, err, "should not error on remove")

				newDev, err = storage.Device(v.setAddr)
				assert.NoError(t, err, "should not error when getting device after removal")
				assert.Equal(t, map[string]float64(nil), newDev.Modes, "modes should be nil after removal")
				assert.Equal(t, false, newDev.Active, "'active' should be false after removal")
			}
		})
	}
}

func TestReportingInterval(t *testing.T) {
	storage := virtualmeter.NewStorage(workdir)
	defer adapterhelper.TearDownAdapter(workdir)[0](t)

	interval := storage.ReportingInterval()
	assert.Equal(t, time.Minute*30, interval, "should default to 30 minutes when nothing set")

	duration := time.Minute * 13

	err := storage.SetReportingInterval(duration)
	assert.NoError(t, err, "should set reporting interval")

	interval = storage.ReportingInterval()
	assert.Equal(t, duration, interval, "should return what was set above")
}
