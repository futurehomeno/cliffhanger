package config_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/config"
)

func TestDefault_Migrate(t *testing.T) {
	t.Parallel()

	t.Run("no migrations provided is a no-op", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Default{ConfigVersion: 1}

		applied, err := cfg.Migrate()

		assert.NoError(t, err)
		assert.Equal(t, 0, applied)
		assert.Equal(t, 1, cfg.ConfigVersion)
	})

	t.Run("current version does not match any migration is a no-op", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Default{ConfigVersion: 99}

		applied, err := cfg.Migrate(
			config.Migration{From: 0, To: 1, Do: func() error { return nil }},
			config.Migration{From: 1, To: 2, Do: func() error { return nil }},
		)

		assert.NoError(t, err)
		assert.Equal(t, 0, applied)
		assert.Equal(t, 99, cfg.ConfigVersion)
	})

	t.Run("applies a single migration from empty version", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Default{}
		called := 0

		applied, err := cfg.Migrate(
			config.Migration{From: 0, To: 1, Do: func() error { called++; return nil }},
		)

		assert.NoError(t, err)
		assert.Equal(t, 1, applied)
		assert.Equal(t, 1, called)
		assert.Equal(t, 1, cfg.ConfigVersion)
	})

	t.Run("chains migrations in order", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Default{}

		var order []string

		applied, err := cfg.Migrate(
			config.Migration{From: 0, To: 1, Do: func() error { order = append(order, "a"); return nil }},
			config.Migration{From: 1, To: 2, Do: func() error { order = append(order, "b"); return nil }},
			config.Migration{From: 2, To: 3, Do: func() error { order = append(order, "c"); return nil }},
		)

		assert.NoError(t, err)
		assert.Equal(t, 3, applied)
		assert.Equal(t, 3, cfg.ConfigVersion)
		assert.Equal(t, []string{"a", "b", "c"}, order)
	})

	t.Run("resumes chain from current version", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Default{ConfigVersion: 2}
		called := 0

		applied, err := cfg.Migrate(
			config.Migration{From: 0, To: 1, Do: func() error { t.Fatal("v0 should not run"); return nil }},
			config.Migration{From: 1, To: 2, Do: func() error { t.Fatal("v1 should not run"); return nil }},
			config.Migration{From: 2, To: 3, Do: func() error { called++; return nil }},
		)

		assert.NoError(t, err)
		assert.Equal(t, 1, applied)
		assert.Equal(t, 1, called)
		assert.Equal(t, 3, cfg.ConfigVersion)
	})

	t.Run("migration with nil Do only advances version", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Default{}

		applied, err := cfg.Migrate(
			config.Migration{From: 0, To: 1, Do: nil},
		)

		assert.NoError(t, err)
		assert.Equal(t, 1, applied)
		assert.Equal(t, 1, cfg.ConfigVersion)
	})

	t.Run("Do mutations on closed-over config are visible", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Default{LogLevel: "debug"}

		applied, err := cfg.Migrate(
			config.Migration{From: 0, To: 1, Do: func() error {
				cfg.LogLevel = "info"
				cfg.LogFile = "/tmp/app.log"

				return nil
			}},
		)

		assert.NoError(t, err)
		assert.Equal(t, 1, applied)
		assert.Equal(t, "info", cfg.LogLevel)
		assert.Equal(t, "/tmp/app.log", cfg.LogFile)
	})

	t.Run("Do error aborts the chain and preserves applied count", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Default{}
		boom := errors.New("boom")

		applied, err := cfg.Migrate(
			config.Migration{From: 0, To: 1, Do: func() error { return nil }},
			config.Migration{From: 1, To: 2, Do: func() error { return boom }},
			config.Migration{From: 2, To: 3, Do: func() error { t.Fatal("should not run"); return nil }},
		)

		assert.ErrorIs(t, err, boom)
		assert.Equal(t, 1, applied)
		assert.Equal(t, 1, cfg.ConfigVersion)
	})

	t.Run("migration whose To equals From is rejected", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Default{ConfigVersion: 1}

		applied, err := cfg.Migrate(
			config.Migration{From: 1, To: 1, Do: func() error { return nil }},
		)

		assert.Error(t, err)
		assert.Equal(t, 0, applied)
		assert.Equal(t, 1, cfg.ConfigVersion)
	})

	t.Run("picks the first matching migration when duplicates exist", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Default{}
		pick := 0

		applied, err := cfg.Migrate(
			config.Migration{From: 0, To: 1, Do: func() error { pick = 1; return nil }},
			config.Migration{From: 0, To: 2, Do: func() error { pick = 2; return nil }},
		)

		assert.NoError(t, err)
		assert.Equal(t, 1, applied)
		assert.Equal(t, 1, cfg.ConfigVersion)
		assert.Equal(t, 1, pick)
	})
}
