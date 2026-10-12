package adapter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
)

func TestShouldSkipServiceTask(t *testing.T) {
	t.Parallel()

	const topic = "/rt:dev/rn:test_adapter/ad:1/sv:out_bin_switch/ad:1"

	thingWithStatus := func(t *testing.T, status adapter.ConnStatusT) adapter.Thing {
		t.Helper()

		th := mockedadapter.NewThing(t)
		th.On("ConnectivityReport").Return(&adapter.ConnectivityReport{
			ConnectivityDetails: &adapter.ConnectivityDetails{ConnStatus: status},
		})

		return th
	}

	tcs := []struct {
		name     string
		registry func(t *testing.T) adapter.ServiceRegistry
		want     bool
	}{
		{
			name:     "registry that is not a thing registry",
			registry: func(t *testing.T) adapter.ServiceRegistry { t.Helper(); return mockedadapter.NewServiceRegistry(t) },
		},
		{
			name: "topic resolves to no thing",
			registry: func(t *testing.T) adapter.ServiceRegistry {
				t.Helper()

				return mockedadapter.NewAdapter(t).WithThingByTopic(topic, true, nil)
			},
		},
		{
			name: "thing is up",
			registry: func(t *testing.T) adapter.ServiceRegistry {
				t.Helper()

				return mockedadapter.NewAdapter(t).WithThingByTopic(topic, true, thingWithStatus(t, adapter.ConnStatusUp))
			},
		},
		{
			name: "thing is down",
			registry: func(t *testing.T) adapter.ServiceRegistry {
				t.Helper()

				return mockedadapter.NewAdapter(t).WithThingByTopic(topic, true, thingWithStatus(t, adapter.ConnStatusDown))
			},
			want: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := mockedadapter.NewService(t)
			svc.On("Topic").Return(topic).Maybe()

			assert.Equal(t, tc.want, adapter.ShouldSkipServiceTask(tc.registry(t), svc))
		})
	}
}
