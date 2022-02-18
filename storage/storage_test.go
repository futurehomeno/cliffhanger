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

func TestStorage_Load(t *testing.T) { //nolint:paralleltest
	tests := []struct {
		name       string
		path       string
		configName string
		want       *testConfig
		wantErr    bool
	}{
		{
			name:       "base case",
			path:       "../testdata/storage/load/",
			configName: config.Name,
			want: &testConfig{
				SettingA: "A",
				SettingB: "B",
				SettingC: "X",
			},
		},
		{
			name:       "config based on defaults only",
			path:       "../testdata/storage/load_defaults_only/",
			configName: config.Name,
			want: &testConfig{
				SettingA: "X",
				SettingB: "Y",
				SettingC: "Z",
			},
		},
		{
			name:       "fallback to defaults",
			path:       "../testdata/storage/load/",
			configName: "config_fallback.json",
			want: &testConfig{
				SettingA: "X",
				SettingB: "X",
				SettingC: "X",
			},
		},
		{
			name:       "reaching for backup on unmarshalling error",
			path:       "../testdata/storage/load_backup_unmarshalling_error/",
			configName: config.Name,
			want: &testConfig{
				SettingA: "A",
				SettingB: "B",
				SettingC: "C",
			},
		},
		{
			name:       "backup as the only source of truth",
			path:       "../testdata/storage/load_backup_only/",
			configName: config.Name,
			want: &testConfig{
				SettingA: "A",
				SettingB: "B",
				SettingC: "C",
			},
		},
		{
			name:       "no data to read",
			path:       "../testdata/storage/empty_dir/",
			configName: config.Name,
			wantErr:    true,
		},
		{
			name:       "invalid data only",
			path:       "../testdata/storage/load_invalid_data_only/",
			configName: config.Name,
			wantErr:    true,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			cfg := &testConfig{}

			srv := storage.New(cfg, tt.path, tt.configName)

			err := srv.Load()
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, cfg)
		})
	}
}

func TestStorage_Save(t *testing.T) { //nolint:paralleltest
	p := "../testdata/storage/empty_dir/"

	err := os.RemoveAll(path.Join(p, storage.DataDirectory))
	assert.NoError(t, err)

	cfg := &testConfig{
		SettingA: "A",
		SettingB: "B",
		SettingC: "C",
	}

	store := storage.New(cfg, p, config.Name)

	// initial save: no configs and backups persisted on disk yet
	err = store.Save()
	assert.NoError(t, err)

	marshalledCfg, err := json.MarshalIndent(cfg, "", "\t")
	assert.NoError(t, err)

	cfgFile, err := ioutil.ReadFile(path.Join(p, storage.DataDirectory, config.Name))
	assert.NoError(t, err)

	// config.json properly persisted on disk
	assert.Equal(t, marshalledCfg, cfgFile)

	// load config from disk
	err = store.Load()
	assert.NoError(t, err)

	newCfg := &testConfig{
		SettingA: "D",
		SettingB: "E",
		SettingC: "F",
	}

	store.Model().(*testConfig).SettingA = newCfg.SettingA //nolint:forcetypeassert
	store.Model().(*testConfig).SettingB = newCfg.SettingB //nolint:forcetypeassert
	store.Model().(*testConfig).SettingC = newCfg.SettingC //nolint:forcetypeassert

	err = store.Save()
	assert.NoError(t, err)

	marshalledNewCfg, err := json.MarshalIndent(newCfg, "", "\t")
	assert.NoError(t, err)

	cfgFile, err = ioutil.ReadFile(path.Join(p, storage.DataDirectory, config.Name))
	assert.NoError(t, err)

	// config.json properly persisted on disk
	assert.Equal(t, marshalledNewCfg, cfgFile)

	backupCfgFile, err := ioutil.ReadFile(path.Join(p, storage.DataDirectory, config.Name+".bak"))
	assert.NoError(t, err)

	// backup should store the previous version of config.
	assert.Equal(t, marshalledCfg, backupCfgFile)

	err = os.RemoveAll(path.Join(p, storage.DataDirectory))
	assert.NoError(t, err)
}
