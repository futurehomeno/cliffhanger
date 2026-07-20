package bootstrap_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/bootstrap"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/manifest"
	"github.com/futurehomeno/cliffhanger/storage"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
)

type fakeApp struct{}

func (a *fakeApp) GetManifest() (*manifest.Manifest, error) { return nil, nil }
func (a *fakeApp) Configure(any) error                      { return nil }
func (a *fakeApp) Uninstall() error                         { return nil }
func (a *fakeApp) ErrorsReport() ([]string, error)          { return nil, nil }

type testCfg struct{ Name string }

func TestEdgeRouting(t *testing.T) {
	t.Parallel()

	cfg := &testCfg{}
	s := storage.New(cfg, t.TempDir(), "config.json")

	ad := mockedadapter.NewAdapter(t)
	ad.On("Name").Return(fimptype.ResourceNameT("test"))

	routes := bootstrap.EdgeRouting(
		"test_service",
		func() any { return cfg },
		nil,
		lifecycle.New(nil),
		s,
		func() *testCfg { return &testCfg{} },
		nil,
		&fakeApp{},
		ad,
		ad.DestroyAllThings,
		nil,
	)

	assert.NotEmpty(t, routes)
}

func TestEdgeTasks(t *testing.T) {
	t.Parallel()

	tasks := bootstrap.EdgeTasks(&fakeApp{}, lifecycle.New(nil), mockedadapter.NewAdapter(t), time.Minute)

	assert.NotEmpty(t, tasks)
}
