package adapterhelper

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/event"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

type FactoryHelper func(adapter adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error)

func (fn FactoryHelper) Create(adapter adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
	return fn(adapter, publisher, thingState)
}

func PrepareAdapter(
	t *testing.T,
	workDir string,
	mqtt *fimpgo.MqttTransport,
	factory FactoryHelper,
) adapter.Adapter {
	state, err := adapter.NewState(workDir)
	if err != nil {
		t.Fatal(fmt.Errorf("adapter helper: failed to create adapter state: %w", err))
	}

	a := adapter.NewAdapter(mqtt, event.NewManager(), factory, state, "test_adapter", "1")

	return a
}

func PrepareSeededAdapter(
	t *testing.T,
	workDir string,
	mqtt *fimpgo.MqttTransport,
	factory FactoryHelper,
	seeds adapter.ThingSeeds,
) adapter.Adapter {
	a := PrepareAdapter(t, workDir, mqtt, factory)

	err := a.InitializeThings()
	if err != nil {
		t.Fatal(fmt.Errorf("adapter helper: failed to initialize things: %w", err))
	}

	if len(seeds) >= 0 {
		err = a.EnsureThings(seeds)
		if err != nil {
			t.Fatal(fmt.Errorf("adapter helper: failed to ensure things: %w", err))
		}
	}

	return a
}

func TearDownAdapter(path string) []suite.Callback {
	return []suite.Callback{
		func(t *testing.T) {
			pathDB := filepath.Join(path, "data.db")
			path = filepath.Join(path, "data")

			err := os.RemoveAll(path)
			if err != nil {
				t.Fatal(fmt.Errorf("adapter helpers: failed to remove adapter data directory at path %s: %w", path, err))
			}

			err = os.RemoveAll(pathDB)
			if err != nil {
				t.Fatal(fmt.Errorf("adapter helpers: failed to remove adapter data.db file %s: %w", pathDB, err))
			}
		},
	}
}
