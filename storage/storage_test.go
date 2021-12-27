package storage_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/storage"
)

type testConfig struct {
	SettingA string
	SettingB string
	SettingC string
}

func TestStorage_Save(t *testing.T) {
	t.Parallel()

	p := "../testdata/storage/save/"

	err := os.RemoveAll(path.Join(p, storage.DataDirectory))
	assert.NoError(t, err)

	cfg := &testConfig{
		SettingA: "A",
		SettingB: "B",
		SettingC: "C",
	}

	srv := storage.New(cfg, p, config.Name)

	err = srv.Save()
	assert.NoError(t, err)

	body, err := json.MarshalIndent(cfg, "", "\t")
	assert.NoError(t, err)

	file, err := ioutil.ReadFile(path.Join(p, storage.DataDirectory, config.Name))
	assert.NoError(t, err)

	assert.Equal(t, body, file)

	err = os.RemoveAll(path.Join(p, storage.DataDirectory))
	assert.NoError(t, err)
}

func TestStorage_Load(t *testing.T) {
	t.Parallel()

	p := "../testdata/storage/load/"

	cfg := &testConfig{}

	srv := storage.New(cfg, p, config.Name)

	err := srv.Load()
	assert.NoError(t, err)

	expectedCfg := &testConfig{
		SettingA: "A",
		SettingB: "B",
		SettingC: "X",
	}

	assert.Equal(t, expectedCfg, cfg)
}

func TestStorage_Load_Fallback(t *testing.T) {
	t.Parallel()

	p := "../testdata/storage/load/"

	cfg := &testConfig{}

	srv := storage.New(cfg, p, "config_fallback.json")

	err := srv.Load()
	assert.NoError(t, err)

	expectedCfg := &testConfig{
		SettingA: "X",
		SettingB: "X",
		SettingC: "X",
	}

	assert.Equal(t, expectedCfg, cfg)
}

