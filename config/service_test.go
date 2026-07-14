package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/storage"
)

type testConfig struct {
	config.Default

	Name     string
	Secret   string
	Interval string
}

func newTestService(t *testing.T) *config.Service[*testConfig] {
	t.Helper()

	s := storage.New(&testConfig{}, t.TempDir(), "config.json")

	return config.NewService(s, func(c *testConfig) *config.Default { return &c.Default }, nil)
}

func TestService_Update(t *testing.T) {
	t.Parallel()

	srv := newTestService(t)

	err := srv.Update(func(c *testConfig) { c.Name = "test" })

	assert.NoError(t, err)
	assert.Equal(t, "test", config.Get(srv, func(c *testConfig) string { return c.Name }))
	assert.NotEmpty(t, srv.Model().ConfiguredAt, "update should stamp the configuration time")
}

func TestService_Persist(t *testing.T) {
	t.Parallel()

	srv := newTestService(t)

	err := srv.Persist(func(c *testConfig) { c.Name = "cache" })

	assert.NoError(t, err)
	assert.Equal(t, "cache", srv.Model().Name)
	assert.Empty(t, srv.Model().ConfiguredAt, "persist should not stamp the configuration time")
}

func TestService_PublicModel(t *testing.T) {
	t.Parallel()

	srv := newTestService(t)
	assert.NoError(t, srv.Update(func(c *testConfig) { c.Secret = "credentials" }))
	assert.Equal(t, srv.Model(), srv.PublicModel(), "no redactor should expose the full model")

	s := storage.New(&testConfig{}, t.TempDir(), "config.json")
	redacting := config.NewService(s, func(c *testConfig) *config.Default { return &c.Default },
		func(c *testConfig) any { public := *c; public.Secret = ""; return &public })

	assert.NoError(t, redacting.Update(func(c *testConfig) { c.Secret = "credentials" }))

	public, ok := redacting.PublicModel().(*testConfig)
	assert.True(t, ok)
	assert.Empty(t, public.Secret)
	assert.Equal(t, "credentials", redacting.Model().Secret)
}

func TestService_GetDuration(t *testing.T) {
	t.Parallel()

	srv := newTestService(t)

	get := func(c *testConfig) string { return c.Interval }

	assert.Equal(t, time.Minute, config.GetDuration(srv, get, time.Minute), "unset value should fall back to default")

	assert.NoError(t, srv.Update(func(c *testConfig) { c.Interval = "30s" }))
	assert.Equal(t, 30*time.Second, config.GetDuration(srv, get, time.Minute))

	assert.NoError(t, srv.Update(func(c *testConfig) { c.Interval = "invalid" }))
	assert.Equal(t, time.Minute, config.GetDuration(srv, get, time.Minute), "invalid value should fall back to default")
}

func TestService_Migrate(t *testing.T) {
	t.Parallel()

	srv := newTestService(t)

	migrated := 0
	migration := config.Migration{From: 0, To: 1, Do: func() error { migrated++; return nil }}

	assert.NoError(t, srv.Migrate(migration))
	assert.Equal(t, 1, migrated)
	assert.Equal(t, 1, srv.Model().ConfigVersion)

	assert.NoError(t, srv.Migrate(migration))
	assert.Equal(t, 1, migrated, "already applied migration should not run again")
}

func TestService_DefaultStore(t *testing.T) {
	t.Parallel()

	srv := newTestService(t)

	assert.NoError(t, srv.DefaultStore().SetLevel("debug"))
	assert.Equal(t, "debug", srv.Model().LogLevel)
}

func TestBackoffConfig_Stateful(t *testing.T) {
	t.Parallel()

	b := config.BackoffConfig{
		Initial:       "1s",
		Repeated:      "2s",
		Final:         "3s",
		InitialCount:  1,
		RepeatedCount: 1,
	}.Stateful(config.BackoffConfig{})

	assert.Equal(t, time.Second, b.Next())
	assert.Equal(t, 2*time.Second, b.Next())
	assert.Equal(t, 3*time.Second, b.Next())

	def := config.BackoffConfig{Initial: "1m", Repeated: "5m", Final: "10m", InitialCount: 1, RepeatedCount: 1}
	fallback := config.BackoffConfig{Initial: "invalid"}.Stateful(def)

	assert.Equal(t, time.Minute, fallback.Next(), "unset or unparsable config should fall back to defaults")
	assert.Equal(t, 5*time.Minute, fallback.Next())
	assert.Equal(t, 10*time.Minute, fallback.Next())
}
