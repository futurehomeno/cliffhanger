package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sync"

	log "github.com/sirupsen/logrus"
)

// Constants defining internal settings of the storage.
const (
	dataDirectory     = "data"
	defaultsDirectory = "defaults"
	backupExtension   = ".bak"

	configFileMode os.FileMode = 0o644
	secretFileMode os.FileMode = 0o640
)

// Storage is an interface representing a service responsible for loading JSON configuration from provided location.
type Storage[T any] interface {
	Load() error
	Save() error
	Reset() error
	Model() T
}

// New creates a new storage service in accordance to Thingsplex layout. Provided model should be a pointer.
func New[T any](model T, workDir string, name string) Storage[T] {
	return &storage[T]{
		lock:         &sync.Mutex{},
		dataPath:     filepath.Join(workDir, dataDirectory, name),
		backupPath:   filepath.Join(workDir, dataDirectory, name) + backupExtension,
		defaultsPath: filepath.Join(workDir, defaultsDirectory, name),
		model:        model,
	}
}

// NewSecrets creates a storage service for credentials and other secrets, conventionally
// data/secrets.json, written with 0640 permissions unlike the world-readable configuration
// and without a defaults file.
func NewSecrets[T any](model T, workDir string, name string) Storage[T] {
	return &storage[T]{
		lock:       &sync.Mutex{},
		dataPath:   filepath.Join(workDir, dataDirectory, name),
		backupPath: filepath.Join(workDir, dataDirectory, name) + backupExtension,
		model:      model,
		mode:       secretFileMode,
		isSecret:   true,
	}
}

// NewCanonicalSecrets creates a secrets storage service following the canonical layout of core applications.
func NewCanonicalSecrets[T any](model T, workDir string, name string) Storage[T] {
	return &storage[T]{
		lock:       &sync.Mutex{},
		dataPath:   filepath.Join(workDir, name),
		backupPath: filepath.Join(workDir, name) + backupExtension,
		model:      model,
		mode:       secretFileMode,
		isSecret:   true,
	}
}

// NewCanonical creates a new storage service allowing canonical separate paths for defaults and data. Provided model should be a pointer.
func NewCanonical[T any](model T, workDir, defaultsDir, name string) Storage[T] {
	return &storage[T]{
		lock:         &sync.Mutex{},
		dataPath:     filepath.Join(workDir, name),
		backupPath:   filepath.Join(workDir, name) + backupExtension,
		defaultsPath: filepath.Join(defaultsDir, name),
		model:        model,
	}
}

func NewState[T any](model T, workDir, name string) Storage[T] {
	return &storage[T]{
		lock:         &sync.Mutex{},
		dataPath:     filepath.Join(workDir, dataDirectory, name),
		backupPath:   filepath.Join(workDir, dataDirectory, name) + backupExtension,
		defaultsPath: "",
		model:        model,
	}
}

func NewCanonicalState[T any](model T, workDir, name string) Storage[T] {
	return &storage[T]{
		lock:         &sync.Mutex{},
		dataPath:     filepath.Join(workDir, name),
		backupPath:   filepath.Join(workDir, name) + backupExtension,
		defaultsPath: "",
		model:        model,
	}
}

// storage is an implementation of the storage service.
type storage[T any] struct {
	lock         *sync.Mutex
	dataPath     string
	backupPath   string
	defaultsPath string
	model        T
	mode         os.FileMode
	isSecret     bool
}

func (s *storage[T]) fileMode() os.FileMode {
	if s.mode == 0 {
		return configFileMode
	}

	return s.mode
}

func (s *storage[T]) Model() T {
	return s.model
}

func (s *storage[T]) Load() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	dataExists, err := s.fileExists(s.dataPath)
	if err != nil {
		return err
	}

	if !dataExists {
		// A crash between the backup rename and the rewrite that follows it in save() leaves only
		// the backup behind; loadData() picks it up, but only if we get that far.
		dataExists, err = s.fileExists(s.backupPath)
		if err != nil {
			return err
		}
	}

	var defaultsExists bool
	if s.defaultsPath != "" {
		defaultsExists, err = s.fileExists(s.defaultsPath)
		if err != nil {
			return err
		}
	}

	if !dataExists && !defaultsExists && s.defaultsPath != "" {
		return fmt.Errorf(
			"storage: no configuration files were found at paths: %s, %s",
			s.dataPath, s.defaultsPath,
		)
	}

	return s.load(defaultsExists, dataExists)
}

func (s *storage[T]) load(defaultsExists, dataExists bool) error {
	if !dataExists && !defaultsExists && s.defaultsPath == "" {
		return nil
	}

	// Always try to load default configuration first.
	if defaultsExists {
		err := s.loadFile(s.defaultsPath)
		if err != nil {
			return err
		}

		if !dataExists {
			return nil
		}
	}

	// Load actual data file.
	err := s.loadData()
	if err != nil {
		if !defaultsExists {
			return err
		}

		log.Errorf("[storage] Read %s, fall back to defaults. err: %v", s.dataPath, err)
	}

	return nil
}

// loadData loads the configuration files and performs fallback to a last backup if possible.
func (s *storage[T]) loadData() error {
	err := s.loadFile(s.dataPath)
	if err == nil {
		return nil
	}

	backupExists, existsErr := s.fileExists(s.backupPath)
	if existsErr != nil {
		return existsErr
	}

	if !backupExists {
		return err
	}

	if backupErr := s.loadFile(s.backupPath); backupErr != nil {
		return backupErr
	}

	log.Errorf("[storage] Read %s, fall back to backup. err: %v", s.dataPath, err)

	return nil
}

// Save saves configuration to the configured location.
func (s *storage[T]) Save() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.save()
}

func (s *storage[T]) save() error {
	err := os.MkdirAll(path.Dir(s.dataPath), 0o755) //nolint:gosec
	if err != nil {
		return fmt.Errorf("storage: cannot create a configuration directory at path %s: %w", path.Dir(s.dataPath), err)
	}

	body, err := json.MarshalIndent(s.model, "", "\t")
	if err != nil {
		return fmt.Errorf("storage: cannot marshal a configuration file at path %s: %w", s.dataPath, err)
	}

	err = s.makeBackup()
	if err != nil {
		return fmt.Errorf("storage: failed to make a configuration backup: %w", err)
	}

	err = s.writeFile(s.dataPath, body)
	if err != nil {
		return fmt.Errorf("storage: cannot save a configuration file at path %s: %w", s.dataPath, err)
	}

	return nil
}

func (s *storage[T]) makeBackup() error {
	cfgExists, err := s.fileExists(s.dataPath)
	if err != nil {
		return err
	}

	if !cfgExists {
		return nil
	}

	// Rename rather than copy: a copy costs a second full write and fsync of the whole file on every
	// Save. loadData() falls back to the backup, which covers a crash between the rename and the
	// rewrite that follows it.
	if err := os.Rename(s.dataPath, s.backupPath); err != nil {
		return fmt.Errorf("storage: failed to move the configuration file at path %s to a backup: %w", s.dataPath, err)
	}

	// The rename carries the old file's mode over, which writeFile used to correct on the way in.
	// A secrets file that pre-dated the 0640 mode would otherwise leave its credentials in a
	// world-readable .bak until the next save renamed a tightened file over it.
	if err := os.Chmod(s.backupPath, s.fileMode()); err != nil {
		return fmt.Errorf("storage: failed to set permissions of the backup file at path %s: %w", s.backupPath, err)
	}

	return nil
}

// Reset drops the stored data and reloads the defaults file. Fields the defaults cannot carry,
// json:"-" ones in particular, are left zeroed for the caller to restore.
func (s *storage[T]) Reset() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	defaultsExists, err := s.fileExists(s.defaultsPath)
	if err != nil {
		return err
	}

	if !defaultsExists && s.defaultsPath != "" {
		return fmt.Errorf("storage: cannot reset as the default configuration file at path %s is not found", s.defaultsPath)
	}

	err = s.removeFile(s.dataPath)
	if err != nil {
		return err
	}

	err = s.removeFile(s.backupPath)
	if err != nil {
		return err
	}

	if s.isSecret {
		// Secrets stores have no defaults to reload from; clear the in-memory model so a reset
		// (logout) does not keep serving stale credentials.
		s.zeroModel()

		return nil
	}

	// A no-defaults state store (adapter.json, config caches) keeps its model: there is nothing to
	// reload and callers hold on to it past the reset.
	if s.defaultsPath == "" {
		return nil
	}

	// Zero before reloading: loadFile unmarshals into the existing model, so fields and map entries
	// absent from the defaults would survive the reset and be written back by the next Save().
	s.zeroModel()

	return s.load(true, false)
}

// zeroModel clears the in-memory model. Zeroes the pointed-to struct in place rather than nilling
// the reference: that wipes the fields any cached pointer still holds and keeps s.model a valid
// (non-nil) target for a later Load().
func (s *storage[T]) zeroModel() {
	if v := reflect.ValueOf(s.model); v.Kind() == reflect.Pointer && !v.IsNil() {
		v.Elem().Set(reflect.Zero(v.Elem().Type()))

		return
	}

	var zero T

	s.model = zero
}

func (s *storage[T]) fileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("storage: cannot verify existence of a configuration file at path %s: %w", path, err)
	}

	return !info.IsDir(), nil
}

func (s *storage[T]) removeFile(path string) error {
	cfgExists, err := s.fileExists(path)
	if err != nil {
		return err
	}

	if cfgExists {
		err = os.Remove(path)
		if err != nil {
			return fmt.Errorf("storage: failed to remove the configuration file at path %s: %w", path, err)
		}
	}

	return nil
}

func (s *storage[T]) loadFile(path string) error {
	body, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return fmt.Errorf("storage: cannot load a configuration file from path %s: %w", path, err)
	}

	err = json.Unmarshal(body, s.model)
	if err != nil {
		return fmt.Errorf("storage: cannot unmarshal a configuration file from path %s with contents '%s': %w", path, body, err)
	}

	return nil
}

func (s *storage[T]) writeFile(path string, data []byte) (err error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, s.fileMode()) //nolint:gosec
	if err != nil {
		return err
	}

	defer func() {
		closeErr := file.Close()

		// if any of the operations below defer fails, we should return that error
		if err != nil {
			return
		}

		err = closeErr
	}()

	// Enforce permissions also on files created before this mode was configured, regardless of umask.
	if err = file.Chmod(s.fileMode()); err != nil {
		return err
	}

	if _, err = file.Write(data); err != nil {
		return
	}

	err = file.Sync()

	return
}
