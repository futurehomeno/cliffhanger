package database_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/database"
)

func TestDatabase_Set_Get(t *testing.T) {
	db := database.NewDomainDatabase("test_domain", makeTestDatabase(t, true))

	assert.NoError(t, db.Set("test_bucket", "test_key", "test_value"))
	assert.NoError(t, db.Stop())

	db = database.NewDomainDatabase("test_domain", makeTestDatabase(t, false))

	var value1, value2 string

	ok, err := db.Get("test_bucket", "test_key", &value1)

	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "test_value", value1)

	ok, err = db.Get("test_bucket", "non_existent_key", &value2)

	assert.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, "", value2)
}

func TestDatabase_SetWithExpiry_Get(t *testing.T) {
	db := database.NewDomainDatabase("test_domain", makeTestDatabase(t, true))

	assert.NoError(t, db.SetWithExpiry("test_bucket", "test_key", "test_value", 1*time.Second))
	assert.NoError(t, db.Stop())

	db = database.NewDomainDatabase("test_domain", makeTestDatabase(t, false))

	var value1, value2 string

	ok, err := db.Get("test_bucket", "test_key", &value1)

	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "test_value", value1)

	time.Sleep(1 * time.Second)

	ok, err = db.Get("test_bucket", "test_key", &value2)

	assert.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, "", value2)
}

func TestDatabase_Keys(t *testing.T) {
	db := database.NewDomainDatabase("test_domain", makeTestDatabase(t, true))

	assert.NoError(t, db.Set("test_bucket", "test_key1", "test_value1"))
	assert.NoError(t, db.Set("test_bucket", "test_key2", "test_value2"))

	keys, err := db.Keys("test_bucket")

	assert.NoError(t, err)
	assert.Equal(t, []string{"test_key1", "test_key2"}, keys)
}

func TestDatabase_KeysBetween(t *testing.T) {
	db := database.NewDomainDatabase("test_domain", makeTestDatabase(t, true))

	assert.NoError(t, db.Set("test_bucket", "test_key1", "test_value1"))
	assert.NoError(t, db.Set("test_bucket", "test_key2", "test_value2"))
	assert.NoError(t, db.Set("test_bucket", "test_key3", "test_value2"))
	assert.NoError(t, db.Set("test_bucket", "test_key4", "test_value2"))
	assert.NoError(t, db.Set("test_bucket", "test_key5", "test_value2"))
	assert.NoError(t, db.Set("test_bucket", "test_key6", "test_value2"))

	keys, err := db.KeysBetween("test_bucket", "test_key2", "test_key5")

	assert.NoError(t, err)
	assert.Equal(t, []string{"test_key2", "test_key3", "test_key4"}, keys)
}

func TestDatabase_Reset(t *testing.T) {
	db := database.NewDomainDatabase("test_domain", makeTestDatabase(t, true))

	assert.NoError(t, db.Set("test_bucket", "test_key1", "test_value1"))
	assert.NoError(t, db.Set("test_bucket", "test_key2", "test_value2"))
	assert.NoError(t, db.Reset())

	keys, err := db.Keys("test_bucket")

	assert.NoError(t, err)
	assert.Equal(t, ([]string)(nil), keys)
}

func TestDatabase_Recovery(t *testing.T) {
	db := database.NewDomainDatabase("test_domain", makeTestDatabase(t, true))

	assert.NoError(t, db.Set("test_bucket", "test_key1", "test_value1"))
	assert.NoError(t, db.Set("test_bucket", "test_key2", "test_value2"))
	assert.NoError(t, db.Set("test_bucket", "test_key3", "test_value2"))
	assert.NoError(t, db.Set("test_bucket", "test_key4", "test_value2"))
	assert.NoError(t, db.Set("test_bucket", "test_key5", "test_value2"))
	assert.NoError(t, db.Set("test_bucket", "test_key6", "test_value2"))
	assert.NoError(t, db.Stop())

	file, err := os.ReadFile("../testdata/database/test.db")

	assert.NoError(t, err)

	// We corrupt the file by removing last two bytes from the file and appending two new lines.
	file = file[0 : len(file)-2]
	file = append(file, []byte("\n\n")...)

	assert.NoError(t, os.WriteFile("../testdata/database/test.db", file, 0644))

	db = database.NewDomainDatabase("test_domain", makeTestDatabase(t, false))

	keys, err := db.Keys("test_bucket")

	assert.NoError(t, err)
	assert.Equal(t, []string{"test_key1", "test_key2", "test_key3", "test_key4", "test_key5"}, keys)
}

func makeTestDatabase(t *testing.T, cleanup bool) database.Database {
	t.Helper()

	if cleanup {
		_ = os.RemoveAll("../testdata/database")
	}

	db, err := database.NewDatabase(
		"../testdata/database",
		database.WithFilename("test"),
		database.WithCompactionPercentage(50),
		database.WithCompactionSize(100*1024),
	)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	err = db.Start()
	if err != nil {
		t.Fatalf("failed to start database: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Stop()

		_ = os.RemoveAll("../testdata/database")
	})

	return db
}
