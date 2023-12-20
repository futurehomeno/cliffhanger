package virtualmeter_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/virtualmeter"
	"github.com/futurehomeno/cliffhanger/database"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
)

func TestVirtualMeterManager_Add(t *testing.T) { //nolint:paralleltest
	cases := []struct {
		name           string
		addr           string
		findThing      bool
		registerDevice bool
		expectError    bool
	}{
		{
			name:           "should not error and should update thing when device added first time",
			addr:           "test",
			registerDevice: true,
			findThing:      true,
			expectError:    false,
		},
		{
			name:           "should return error when no thing is registered",
			addr:           "test",
			registerDevice: false,
			findThing:      true,
			expectError:    true,
		},
		{
			name:           "should error when thing isn't found",
			addr:           "test",
			registerDevice: true,
			findThing:      false,
			expectError:    true,
		},
	}

	for _, cc := range cases {
		c := cc
		t.Run(c.name, func(t *testing.T) {
			db, _ := database.NewDatabase(workdir)

			manager := virtualmeter.NewVirtualMeterManager(db, time.Second)
			thing := &mockedadapter.Thing{}
			ad := &mockedadapter.Adapter{}

			defer adapterhelper.TearDownAdapter(workdir)[0](t)

			manager.WithAdapter(ad)

			if c.findThing {
				ad.On("ThingByAddress", c.addr).Return(thing)
			} else {
				ad.On("ThingByAddress", c.addr).Return(nil)
			}

			if c.registerDevice {
				err := manager.RegisterDevice(thing, c.addr, nil,
					numericmeter.Specification("", "", "", "", nil, nil),
				)
				assert.NoError(t, err)
			}

			if c.findThing && c.registerDevice {
				thing.On("Update", true, mock.AnythingOfType("adapter.ThingUpdate")).Return(nil)
			}

			modes := map[string]float64{"on": 432}
			unit := "W"
			err := manager.Add(c.addr, modes, unit)

			if c.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err, "error isn't expected")

				assert.NoError(t, err, "getting device shouldn't error")
				newModes, _ := manager.Modes(c.addr)
				assert.Equal(t, modes, newModes)
			}

			ad.AssertExpectations(t)
			thing.AssertExpectations(t)
		})
	}
}
