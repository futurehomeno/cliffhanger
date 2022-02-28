package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"

	log "github.com/sirupsen/logrus"
)

// Constants defining data and defaults locations.
const (
	DataDirectory     = "data"
	DefaultsDirectory = "defaults"

	backupExtension = ".bak"
)

// Storage is an interface representing a service responsible for loading JSON configuration from provided location.
type Storage interface {
	// Load loads the configuration. First a default configuration is loaded if present and then an actual configuration is used to override the defaults.
	Load() error
	// Save saves configuration to the configured location.
	Save() error
	// Model returns a configuration model object.
	Model() interface{}
}

// New creates new storage service. Provided config model should be a pointer.
func New(cfg interface{}, workDir string, name string) Storage {
	return &storage{
		lock:    &sync.Mutex{},
		workDir: workDir,
		name:    name,
		config:  cfg,
	}
}

// storage is an implementation of the storage service.
type storage struct {
	lock    *sync.Mutex
	workDir string
	name    string
	config  interface{}
}

// Model returns a configuration model object.
func (s *storage) Model() interface{} {
	return s.config
}

// Load loads the configuration. First a default configuration is loaded if present and then an actual configuration is used to override the defaults.
func (s *storage) Load() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	dataExists, err := s.fileExists(s.getDataPath())
	if err != nil {
		return err
	}

	defaultsExists, err := s.fileExists(s.getDefaultPath())
	if err != nil {
		return err
	}

	if !dataExists && !defaultsExists {
		return fmt.Errorf(
			"storage: no configuration files were found at paths: %s, %s",
			s.getDataPath(), s.getDefaultPath(),
		)
	}

	return s.load(defaultsExists, dataExists)
}

// load loads the configuration files in the right order and performs fallback if allowed.
func (s *storage) load(defaultsExists, dataExists bool) error {
	// Always try to load default configuration first.
	if defaultsExists {
		err := s.loadFile(s.getDefaultPath())
		if err != nil {
			return err
		}

		if !dataExists {
			return nil
		}
	}

	// Load actual data file.
	err := s.loadData()
	if err != nil && !defaultsExists {
		return err
	}

	log.WithError(err).Errorf("storage: failed to read the configuration file at path %s, falling back to defaults", s.getDataPath())

	return nil
}

// load loads the configuration files and performs fallback to a last backup if possible.
func (s *storage) loadData() error {
	err := s.loadFile(s.getDataPath())
	if err == nil {
		return nil
	}

	backupExists, existsErr := s.fileExists(s.getBackupPath())
	if existsErr != nil {
		return existsErr
	}

	if !backupExists {
		return err
	}

	err = s.loadFile(s.getBackupPath())
	if err != nil {
		return err
	}

	log.WithError(err).Errorf("storage: failed to read the configuration file at path %s, falling back to last backup", s.getDataPath())

	return nil
}

// Save saves configuration to the configured location.
func (s *storage) Save() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := os.MkdirAll(path.Dir(s.getDataPath()), 0774) //nolint:gofumpt
	if err != nil {
		return fmt.Errorf("storage: cannot create a configuration directory at path %s: %w", path.Dir(s.getDataPath()), err)
	}

	err = s.makeBackup()
	if err != nil {
		return fmt.Errorf("storage: failed to make a configuration backup: %w", err)
	}

	body, err := json.MarshalIndent(s.config, "", "\t")
	if err != nil {
		return fmt.Errorf("storage: cannot marshal a configuration file at path %s: %w", s.getDataPath(), err)
	}

	//nolint:gosec
	err = ioutil.WriteFile(s.getDataPath(), body, 0664) //nolint:gofumpt
	if err != nil {
		return fmt.Errorf("storage: cannot save a configuration file at path %s: %w", s.getDataPath(), err)
	}

	return nil
}

// makeBackup copies contents of an existing configuration to a backup file.
func (s *storage) makeBackup() error {
	cfgExists, err := s.fileExists(s.getDataPath())
	if err != nil {
		return err
	}

	if !cfgExists {
		return nil
	}

	body, err := ioutil.ReadFile(s.getDataPath())
	if err != nil {
		return err
	}

	//nolint:gosec
	err = ioutil.WriteFile(s.getBackupPath(), body, 0664) //nolint:gofumpt
	if err != nil {
		return err
	}

	return nil
}

// getDataPath returns the data path.
func (s *storage) getDataPath() string {
	return filepath.Join(s.workDir, DataDirectory, s.name)
}

// getDefaultPath returns the defaults path.
func (s *storage) getDefaultPath() string {
	return filepath.Join(s.workDir, DefaultsDirectory, s.name)
}

// getBackupPath returns the backup path.
func (s *storage) getBackupPath() string {
	return s.getDataPath() + backupExtension
}

// fileExists checks if the file exists.
func (s *storage) fileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("storage: cannot verify existence of a configuration file at path %s: %w", path, err)
	}

	return !info.IsDir(), nil
}

// loadFile loads a provided file and unmarshalls it using the configured model.
func (s *storage) loadFile(path string) error {
	body, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("storage: cannot load a configuration file from path %s: %w", path, err)
	}

	err = json.Unmarshal(body, s.config)
	if err != nil {
		return fmt.Errorf("storage: cannot unmarshal a configuration file from path %s with contents '%s': %w", path, body, err)
	}

	return nil
}
