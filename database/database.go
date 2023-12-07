package database

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/tidwall/buntdb"
)

// Database is a simple in-memory key-value store with append-only persistence to disk to minimize the disk IO and risk of data corruption.
type Database interface {
	Start() error
	Stop() error
	Reset() error

	Keys(bucket string) ([]string, error)
	KeysBetween(bucket, from, to string) ([]string, error)
	Get(bucket, key string, value interface{}) (ok bool, err error)
	Set(bucket, key string, value interface{}) error
	SetWithExpiry(bucket, key string, value interface{}, expiry time.Duration) error
}

func NewDatabase(workdir string, options ...Option) (Database, error) {
	cfg := newConfig().withDefaults().apply(append(options, WithWorkdir(workdir))...)

	db, err := prepareDatabase(cfg.workdir, cfg.filename)
	if err != nil {
		return nil, fmt.Errorf("database: failed to prepare database: %w", err)
	}

	err = db.SetConfig(buntdb.Config{
		SyncPolicy:           buntdb.Always,
		AutoShrinkPercentage: cfg.compactionPercentage,
		AutoShrinkMinSize:    cfg.compactionSize,
		AutoShrinkDisabled:   false,
	})
	if err != nil {
		return nil, fmt.Errorf("database: failed to set database config: %w", err)
	}

	return &database{
		db: db,
	}, nil
}

func prepareDatabase(workdir, filename string) (*buntdb.DB, error) {
	err := os.MkdirAll(workdir, 0774)
	if err != nil {
		return nil, fmt.Errorf("database: failed to create work directory: %w", err)
	}

	db, err := buntdb.Open(path.Join(workdir, filename+".db"))
	if err != nil {
		if errors.Is(err, buntdb.ErrInvalid) || errors.Is(err, io.ErrUnexpectedEOF) {
			return recoverDatabase(workdir, filename)
		}

		return nil, fmt.Errorf("databse: failed to open data file: %w", err)
	}

	return db, nil
}

func recoverDatabase(workdir, filename string) (*buntdb.DB, error) {
	corruptedData, err := os.ReadFile(path.Join(workdir, filename+".db"))
	if err != nil {
		return nil, fmt.Errorf("database: failed to read corrupted data file: %w", err)
	}

	tempDb, err := buntdb.Open(":memory:")
	if err != nil {
		return nil, fmt.Errorf("database: failed to open temporary database: %w", err)
	}

	_ = tempDb.Load(bytes.NewReader(corruptedData))

	f, err := os.OpenFile(path.Join(workdir, filename+".db.recovered"), os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("database: failed to create recovered data file: %w", err)
	}

	err = tempDb.Save(f)
	if err != nil {
		return nil, fmt.Errorf("database: failed to save recovered data file: %w", err)
	}

	err = tempDb.Close()
	if err != nil {
		return nil, fmt.Errorf("database: failed to close temporary database: %w", err)
	}

	err = f.Close()
	if err != nil {
		return nil, fmt.Errorf("database: failed to close recovered data file: %w", err)
	}

	err = os.Remove(path.Join(workdir, filename+".db.corrupted"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("database: failed to remove previous corrupted data file: %w", err)
	}

	err = os.Rename(path.Join(workdir, filename+".db"), path.Join(workdir, filename+".db.corrupted"))
	if err != nil {
		return nil, fmt.Errorf("database: failed to rename corrupted data file: %w", err)
	}

	err = os.Rename(path.Join(workdir, filename+".db.recovered"), path.Join(workdir, filename+".db"))
	if err != nil {
		return nil, fmt.Errorf("database: failed to rename recovered data file: %w", err)
	}

	db, err := buntdb.Open(path.Join(workdir, filename+".db"))
	if err != nil {
		return nil, fmt.Errorf("database: failed to open recovered data file: %w", err)
	}

	return db, nil
}

type database struct {
	db *buntdb.DB
}

func (d *database) Start() error {
	return nil
}

func (d *database) Stop() error {
	err := d.db.Close()
	if err != nil {
		return fmt.Errorf("database: failed to close the database: %w", err)
	}

	return nil
}

func (d *database) Reset() error {
	err := d.db.Update(func(tx *buntdb.Tx) error {
		return tx.DeleteAll()
	})
	if err != nil {
		return fmt.Errorf("database: failed to delete key from the database: %w", err)
	}

	err = d.db.Shrink()
	if err != nil {
		return fmt.Errorf("database: failed to shrink the database: %w", err)
	}

	return nil
}

func (d *database) Set(bucket, key string, value interface{}) error {
	return d.SetWithExpiry(bucket, key, value, 0)
}

func (d *database) SetWithExpiry(bucket, key string, value interface{}, expiry time.Duration) error {
	return d.db.Update(func(tx *buntdb.Tx) error {
		rawData, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("database: failed to marshal value: %w", err)
		}

		var options *buntdb.SetOptions
		if expiry > 0 {
			options = &buntdb.SetOptions{
				Expires: true,
				TTL:     expiry,
			}
		}

		_, _, err = tx.Set(d.key(bucket, key), string(rawData), options)
		if err != nil {
			return fmt.Errorf("database: failed to set the key: %w", err)
		}

		return nil
	})
}

func (d *database) Get(bucket, key string, value interface{}) (ok bool, err error) {
	err = d.db.View(func(tx *buntdb.Tx) error {
		rawData, txErr := tx.Get(d.key(bucket, key))
		if txErr != nil {
			return txErr
		}

		return json.Unmarshal([]byte(rawData), value)
	})
	if err != nil && !errors.Is(err, buntdb.ErrNotFound) {
		return false, fmt.Errorf("database: failed to get the key: %w", err)
	}

	return !errors.Is(err, buntdb.ErrNotFound), nil
}

func (d *database) Keys(bucket string) ([]string, error) {
	var keys []string

	err := d.db.View(func(tx *buntdb.Tx) error {
		return tx.AscendKeys(fmt.Sprintf("%s:*", bucket), func(key, value string) bool {
			keys = append(keys, key)

			return true
		})
	})
	if err != nil {
		return nil, fmt.Errorf("database: failed to get the keys for bucket %s: %w", bucket, err)
	}

	for i, key := range keys {
		keys[i] = key[len(bucket)+1:]
	}

	return keys, nil
}

func (d *database) KeysBetween(bucket, from, to string) ([]string, error) {
	var keys []string

	err := d.db.View(func(tx *buntdb.Tx) error {
		return tx.AscendRange("", fmt.Sprintf("%s:%s", bucket, from), fmt.Sprintf("%s:%s", bucket, to), func(key, value string) bool {
			keys = append(keys, key)

			return true
		})
	})
	if err != nil {
		return nil, fmt.Errorf("database: failed to get the keys for bucket %s between %s and %s: %w", bucket, from, to, err)
	}

	for i, key := range keys {
		keys[i] = key[len(bucket)+1:]
	}

	return keys, nil
}

func (d *database) key(bucket, key string) string {
	return fmt.Sprintf("%s:%s", bucket, key)
}

func NewDomainDatabase(domain string, db Database) Database {
	return &domainDatabase{
		domain:   domain,
		Database: db,
	}
}

type domainDatabase struct {
	Database

	domain string
}

func (d *domainDatabase) Keys(bucket string) ([]string, error) {
	return d.Database.Keys(d.bucket(bucket))
}

func (d *domainDatabase) KeysBetween(bucket, from, to string) ([]string, error) {
	return d.Database.KeysBetween(d.bucket(bucket), from, to)
}

func (d *domainDatabase) Get(bucket, key string, value interface{}) (bool, error) {
	return d.Database.Get(d.bucket(bucket), key, value)
}

func (d *domainDatabase) Set(bucket, key string, value interface{}) error {
	return d.Database.Set(d.bucket(bucket), key, value)
}

func (d *domainDatabase) SetWithExpiry(bucket, key string, value interface{}, expiry time.Duration) error {
	return d.Database.SetWithExpiry(d.bucket(bucket), key, value, expiry)
}

func (d *domainDatabase) bucket(bucket string) string {
	return fmt.Sprintf("%s:%s", d.domain, bucket)
}
