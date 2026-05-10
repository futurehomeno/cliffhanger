package storage_test

import (
	"encoding/json"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/storage"
	"github.com/futurehomeno/cliffhanger/telemetry/types"
)

const (
	backupExtension = ".bak"
	configFileName  = "config.json"
)

type testConfig struct {
	config.Default

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
			configName: configFileName,
			want: &testConfig{
				SettingA: "A",
				SettingB: "B",
				SettingC: "X",
			},
		},
		{
			name:       "model based on defaults only",
			path:       "../testdata/storage/load_defaults_only/",
			configName: configFileName,
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
			configName: configFileName,
			want: &testConfig{
				SettingA: "A",
				SettingB: "B",
				SettingC: "C",
			},
		},
		{
			name:       "no data to read",
			path:       "../testdata/storage/empty_dir/",
			configName: configFileName,
			wantErr:    true,
		},
		{
			name:       "invalid data only",
			path:       "../testdata/storage/load_invalid_data_only/",
			configName: configFileName,
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

	err := os.RemoveAll(path.Join(p, "data"))
	assert.NoError(t, err)

	cfg := &testConfig{
		SettingA: "A",
		SettingB: "B",
		SettingC: "C",
	}

	store := storage.New(cfg, p, configFileName)

	// initial save: no configs and backups persisted on disk yet
	err = store.Save()
	assert.NoError(t, err)

	marshalledCfg, err := json.MarshalIndent(cfg, "", "\t")
	assert.NoError(t, err)

	cfgFile, err := os.ReadFile(path.Join(p, "data", configFileName))
	assert.NoError(t, err)

	// model.json properly persisted on disk
	assert.Equal(t, marshalledCfg, cfgFile)

	// load model from disk
	err = store.Load()
	assert.NoError(t, err)

	newCfg := &testConfig{
		SettingA: "D",
		SettingB: "E",
		SettingC: "F",
	}

	store.Model().SettingA = newCfg.SettingA
	store.Model().SettingB = newCfg.SettingB
	store.Model().SettingC = newCfg.SettingC

	err = store.Save()
	assert.NoError(t, err)

	marshalledNewCfg, err := json.MarshalIndent(newCfg, "", "\t")
	assert.NoError(t, err)

	cfgFile, err = os.ReadFile(path.Join(p, "data", configFileName))
	assert.NoError(t, err)

	// model.json properly persisted on disk
	assert.Equal(t, marshalledNewCfg, cfgFile)

	backupCfgFile, err := os.ReadFile(path.Join(p, "data", configFileName+backupExtension))
	assert.NoError(t, err)

	// backup should store the previous version of model.
	assert.Equal(t, marshalledCfg, backupCfgFile)

	err = os.RemoveAll(path.Join(p, "data"))
	assert.NoError(t, err)
}

func TestStorage_Reset(t *testing.T) { //nolint:paralleltest
	p := "../testdata/storage/reset/"

	configData := []byte(`{"SettingA": "A","SettingB": "B","SettingC": "C"}`)

	err := os.MkdirAll(path.Join(p, "data"), 0755) //nolint:gofumpt,gosec
	assert.NoError(t, err)

	err = os.WriteFile(path.Join(p, "data", configFileName+backupExtension), configData, 0664) //nolint:gofumpt,gosec
	assert.NoError(t, err)

	err = os.WriteFile(path.Join(p, "data", configFileName), configData, 0664) //nolint:gofumpt,gosec
	assert.NoError(t, err)

	store := storage.New(&testConfig{}, p, configFileName)

	err = store.Load()
	assert.NoError(t, err)
	assert.Equal(t, &testConfig{SettingA: "A", SettingB: "B", SettingC: "C"}, store.Model())

	err = store.Reset()
	assert.NoError(t, err)
	assert.Equal(t, &testConfig{SettingA: "X", SettingB: "X", SettingC: "X"}, store.Model())

	_, err = os.Stat(path.Join(p, "data", configFileName+backupExtension))
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(path.Join(p, "data", configFileName))
	assert.True(t, os.IsNotExist(err))
}

func TestStorage_RoundTrip_WithEmbeddedDefault(t *testing.T) { //nolint:paralleltest
	p := "../testdata/storage/empty_dir/"

	require.NoError(t, os.RemoveAll(path.Join(p, "data")))

	t.Cleanup(func() {
		_ = os.RemoveAll(path.Join(p, "data"))
	})

	original := &testConfig{
		Default: config.Default{
			WorkDir:            "/tmp/work",
			ConfigDir:          "/tmp/cfg",
			ConfigVersion:      3,
			MQTTServerURI:      "tcp://broker:1883",
			MQTTUsername:       "user",
			MQTTPassword:       "pass",
			MQTTClientIDPrefix: "prefix",
			InfoFile:           "/var/info",
			LogFile:            "/var/log/app.log",
			LogLevel:           "debug",
			LogFormat:          "json",
			LogRevertTimeout:   2 * time.Hour,
			LogRevertAt:        time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
			RestartsCount:      7,
			Telemetry:          &types.TelemetryConfig{Enabled: true, Validity: time.Hour},
			ConfiguredAt:       "2026-05-10T12:00:00Z",
		},
		SettingA: "a",
		SettingB: "b",
		SettingC: "c",
	}

	require.NoError(t, storage.New(original, p, configFileName).Save())

	loaded := &testConfig{}
	require.NoError(t, storage.New(loaded, p, configFileName).Load())

	// WorkDir and ConfigDir are tagged json:"-" and must not survive the round-trip.
	assert.Empty(t, loaded.WorkDir)
	assert.Empty(t, loaded.ConfigDir)

	// Everything else, including the embedded Telemetry pointer, must round-trip.
	expected := *original
	expected.WorkDir = ""
	expected.ConfigDir = ""
	assert.Equal(t, &expected, loaded)
}

func TestStorage_DefaultConfigIf_ThroughDefaultStoreFromStorage(t *testing.T) { //nolint:paralleltest
	p := "../testdata/storage/empty_dir/"

	require.NoError(t, os.RemoveAll(path.Join(p, "data")))

	t.Cleanup(func() {
		_ = os.RemoveAll(path.Join(p, "data"))
	})

	cfg := &testConfig{SettingA: "a"}
	store := storage.New(cfg, p, configFileName)

	ds := config.NewDefaultStoreFromStorage(store, func(c *testConfig) *config.Default { return &c.Default })

	// GetTelemetry on a freshly-picked Default returns the zero value without error.
	got, err := ds.Default().GetTelemetry()
	require.NoError(t, err)
	assert.Equal(t, types.TelemetryConfig{}, got)

	// IncrementRestartsCount goes through the embedded *Default and persists via storage.Save.
	count, err := ds.IncrementRestartsCount()
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	count, err = ds.IncrementRestartsCount()
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// SetTelemetry copies the input under the lock and persists through storage.
	tc := &types.TelemetryConfig{Enabled: true, Validity: 30 * time.Minute}
	require.NoError(t, ds.SetTelemetry(tc))

	// saveStamped called SetConfiguredAt with sub-second precision (RFC3339Nano).
	require.NotEmpty(t, cfg.ConfiguredAt)
	parsedAt, err := time.Parse(time.RFC3339Nano, cfg.ConfiguredAt)
	require.NoError(t, err, "ConfiguredAt must be RFC3339Nano-formatted")
	assert.WithinDuration(t, time.Now(), parsedAt, time.Minute)

	// Reload from disk into a fresh model — every Default field touched above must come back.
	reloaded := &testConfig{}
	require.NoError(t, storage.New(reloaded, p, configFileName).Load())

	assert.Equal(t, "a", reloaded.SettingA)
	assert.Equal(t, 2, reloaded.RestartsCount)
	require.NotNil(t, reloaded.Telemetry)
	assert.True(t, reloaded.Telemetry.Enabled)
	assert.Equal(t, 30*time.Minute, reloaded.Telemetry.Validity)
	assert.Equal(t, cfg.ConfiguredAt, reloaded.ConfiguredAt)

	// GetTelemetry on the reloaded *Default returns a value-copy of the persisted block.
	gotReloaded, err := reloaded.Default.GetTelemetry()
	require.NoError(t, err)
	assert.Equal(t, *reloaded.Telemetry, gotReloaded)
}

func TestStorage_DefaultStoreFromStorage_PersistsThroughStorage(t *testing.T) { //nolint:paralleltest
	p := "../testdata/storage/empty_dir/"

	require.NoError(t, os.RemoveAll(path.Join(p, "data")))

	t.Cleanup(func() {
		_ = os.RemoveAll(path.Join(p, "data"))
	})

	cfg := &testConfig{SettingA: "a"}
	store := storage.New(cfg, p, configFileName)

	ds := config.NewDefaultStoreFromStorage(store, func(c *testConfig) *config.Default { return &c.Default })

	require.NoError(t, ds.SetLevel("trace"))
	require.NoError(t, ds.SetLogFile("/var/log/app.log"))
	require.NoError(t, ds.SetTelemetry(&types.TelemetryConfig{Enabled: true, Validity: time.Hour}))

	// Reading via the store still sees the live model.
	assert.Equal(t, "trace", ds.Level())
	assert.NotEmpty(t, cfg.ConfiguredAt, "DefaultStore.Save stamps ConfiguredAt on every write")

	// Reload from disk into a fresh model — embedded Default fields must come back.
	reloaded := &testConfig{}
	require.NoError(t, storage.New(reloaded, p, configFileName).Load())

	assert.Equal(t, "a", reloaded.SettingA)
	assert.Equal(t, "trace", reloaded.LogLevel)
	assert.Equal(t, "/var/log/app.log", reloaded.LogFile)
	require.NotNil(t, reloaded.Telemetry)
	assert.True(t, reloaded.Telemetry.Enabled)
	assert.Equal(t, time.Hour, reloaded.Telemetry.Validity)
	assert.Equal(t, cfg.ConfiguredAt, reloaded.ConfiguredAt)
}
