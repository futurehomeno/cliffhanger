package virtualmeter_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter/service/virtualmeter"
	"github.com/futurehomeno/cliffhanger/database"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
)

func TestStorage_Device(t *testing.T) { //nolint:paralleltest
	cases := []struct {
		name         string
		device       *virtualmeter.Device
		setAddr      string
		expectToFind bool
	}{
		{
			name: "should save and find a device",
			device: &virtualmeter.Device{
				Modes: map[string]float64{
					"test": 123,
					"beet": 321,
				},
				CurrentMode:       "mode",
				AccumulatedEnergy: 4321.32123,
				LastTimeUpdated:   time.Now(),
				Unit:              "W",
				Active:            true,
			},
			setAddr:      "test",
			expectToFind: true,
		},
		{
			name: "should not find a device when setting by a wrong key",
			device: &virtualmeter.Device{
				Modes: map[string]float64{
					"test": 123,
					"beet": 321,
				},
			},
			setAddr:      "invalid",
			expectToFind: false,
		},
	}

	for _, vv := range cases { //nolint:paralleltest
		v := vv

		t.Run(v.name, func(t *testing.T) {
			db, _ := database.NewDatabase(workdir)

			storage := virtualmeter.NewStorage(db)

			defer adapterhelper.TearDownAdapter(workdir)[0](t)

			err := storage.SetDevice(v.setAddr, v.device)
			assert.NoError(t, err, "should set a device")

			newDev, err := storage.Device("test")

			assert.NoError(t, err, "shouldn't return errors")

			if v.expectToFind {
				assert.Equal(t, v.device.Modes, newDev.Modes, "should find a device with the same modes")
				assert.Equal(t, v.device.CurrentMode, newDev.CurrentMode, "should find a device with the same mode")
				assert.Equal(t, v.device.AccumulatedEnergy, newDev.AccumulatedEnergy, "should find a device with the same accumulated energy")
				assert.Equal(t, v.device.Unit, newDev.Unit, "should find a device with the same unit")
				assert.Equal(t, v.device.Active, newDev.Active, "should find a device with the same active")
			} else {
				assert.Nil(t, newDev, "should return nil device")
			}

			if v.expectToFind {
				err := storage.CleanDevice(v.setAddr)
				assert.NoError(t, err, "should not error on remove")

				newDev, err = storage.Device(v.setAddr)
				assert.NoError(t, err, "should not error when getting device after removal")
				assert.Equal(t, map[string]float64(nil), newDev.Modes, "modes should be nil after removal")
				assert.Equal(t, false, newDev.Active, "'active' should be false after removal")
			}
		})
	}
}
