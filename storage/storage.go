package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"

	log "github.com/sirupsen/logrus"
)

// Constants defining internal settings of the storage.
const (
	dataDirectory     = "data"
	defaultsDirectory = "defaults"
	backupExtension   = ".bak"
)

// Storage is an interface representing a service responsible for loading JSON configuration from provided location.
type Storage interface {
	// Load loads the configuration. First a default configuration is loaded if present and then an actual configuration is used to override the defaults.
	Load() error
	// Save saves configuration to the configured location.
	Save() error
	// Reset deletes the configuration file and reloads default configuration.
	Reset() error
	// Model returns a configuration model object.
	Model() interface{}
}

// New creates a new storage service in accordance to Thingsplex layout. Provided model should be a pointer.
func New(model interface{}, workDir string, name string) Storage {
	return &storage{
		lock:         &sync.Mutex{},
		dataPath:     filepath.Join(workDir, dataDirectory, name),
		backupPath:   filepath.Join(workDir, dataDirectory, name) + backupExtension,
		defaultsPath: filepath.Join(workDir, defaultsDirectory, name),
		model:        model,
	}
}

// NewCanonical creates a new storage service allowing canonical separate paths for defaults and data. Provided model should be a pointer.
func NewCanonical(model interface{}, workDir, defaultsDir, name string) Storage {
	return &storage{
		lock:         &sync.Mutex{},
		dataPath:     filepath.Join(workDir, name),
		backupPath:   filepath.Join(workDir, name) + backupExtension,
		defaultsPath: filepath.Join(defaultsDir, name),
		model:        model,
	}
}

func NewState(model interface{}, workDir, name string) Storage {
	return &storage{
		lock:         &sync.Mutex{},
		dataPath:     filepath.Join(workDir, dataDirectory, name),
		backupPath:   filepath.Join(workDir, dataDirectory, name) + backupExtension,
		defaultsPath: "",
		model:        model,
	}
}

func NewCanonicalState(model interface{}, workDir, name string) Storage {
	return &storage{
		lock:         &sync.Mutex{},
		dataPath:     filepath.Join(workDir, name),
		backupPath:   filepath.Join(workDir, name) + backupExtension,
		defaultsPath: "",
		model:        model,
	}
}

// storage is an implementation of the storage service.
type storage struct {
	lock         *sync.Mutex
	dataPath     string
	backupPath   string
	defaultsPath string
	model        interface{}
}

// Model returns a configuration model object.
func (s *storage) Model() interface{} {
	return s.model
}

// Load loads the configuration. First a default configuration is loaded if present and then an actual configuration is used to override the defaults.
func (s *storage) Load() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	dataExists, err := s.fileExists(s.dataPath)
	if err != nil {
		return err
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

// load performs loading of the configuration files in the right order and performs fallback if allowed.
func (s *storage) load(defaultsExists, dataExists bool) error {
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

		log.WithError(err).Errorf("storage: failed to read the configuration file at path %s, falling back to defaults", s.dataPath)
	}

	return nil
}

// loadData loads the configuration files and performs fallback to a last backup if possible.
func (s *storage) loadData() error {
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

	err = s.loadFile(s.backupPath)
	if err != nil {
		return err
	}

	log.WithError(err).Errorf("storage: failed to read the configuration file at path %s, falling back to last backup", s.dataPath)

	return nil
}

// Save saves configuration to the configured location.
func (s *storage) Save() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := os.MkdirAll(path.Dir(s.dataPath), 0774) //nolint:gofumpt
	if err != nil {
		return fmt.Errorf("storage: cannot create a configuration directory at path %s: %w", path.Dir(s.dataPath), err)
	}

	err = s.makeBackup()
	if err != nil {
		return fmt.Errorf("storage: failed to make a configuration backup: %w", err)
	}

	body, err := json.MarshalIndent(s.model, "", "\t")
	if err != nil {
		return fmt.Errorf("storage: cannot marshal a configuration file at path %s: %w", s.dataPath, err)
	}

	//nolint:gosec
	err = os.WriteFile(s.dataPath, body, 0664) //nolint:gofumpt
	if err != nil {
		return fmt.Errorf("storage: cannot save a configuration file at path %s: %w", s.dataPath, err)
	}

	return nil
}

// makeBackup copies contents of an existing configuration to a backup file.
func (s *storage) makeBackup() error {
	cfgExists, err := s.fileExists(s.dataPath)
	if err != nil {
		return err
	}

	if !cfgExists {
		return nil
	}

	body, err := os.ReadFile(s.dataPath)
	if err != nil {
		return err
	}

	//nolint:gosec
	err = os.WriteFile(s.backupPath, body, 0664) //nolint:gofumpt
	if err != nil {
		return err
	}

	return nil
}

// Reset deletes the configuration file and reloads default configuration.
func (s *storage) Reset() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	defaultsExists, err := s.fileExists(s.defaultsPath)
	if err != nil {
		return err
	}

	if !defaultsExists && s.defaultsPath != "" {
		return fmt.Errorf("storage: cannot reset as the default configuration file at path %s is not found", s.defaultsPath)
	}

	cfgExists, err := s.fileExists(s.dataPath)
	if err != nil {
		return err
	}

	if cfgExists {
		err = os.Remove(s.dataPath)
		if err != nil {
			return fmt.Errorf("storage: failed to remove the configuration file at path %s: %w", s.dataPath, err)
		}
	}

	backupExists, err := s.fileExists(s.backupPath)
	if err != nil {
		return err
	}

	if backupExists {
		err = os.Remove(s.backupPath)
		if err != nil {
			return fmt.Errorf("storage: failed to remove the configuration file at path %s: %w", s.backupPath, err)
		}
	}

	if s.defaultsPath == "" {
		return nil
	}

	return s.load(true, false)
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
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("storage: cannot load a configuration file from path %s: %w", path, err)
	}

	err = json.Unmarshal(body, s.model)
	if err != nil {
		return fmt.Errorf("storage: cannot unmarshal a configuration file from path %s with contents '%s': %w", path, body, err)
	}

	return nil
}
