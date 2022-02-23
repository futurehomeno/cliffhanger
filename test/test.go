package test

import (
	"github.com/futurehomeno/cliffhanger/storage"
)

// StorageMock represents a storage.Storage with mocking capabilities.
type StorageMock interface {
	storage.Storage

	// MockLoad allows mocking 'Load' method.
	MockLoad(err error)
	// MockSave allows mocking 'Save' method.
	MockSave(err error)
}

type storageMock struct {
	cfg        interface{}
	mockedLoad error
	mockedSave error
}

// NewStorageMock returns a new instance of StorageMock.
func NewStorageMock(cfg interface{}) StorageMock {
	return &storageMock{cfg: cfg}
}

func (m *storageMock) Load() error {
	return m.mockedLoad
}

func (m *storageMock) Save() error {
	return m.mockedSave
}

func (m *storageMock) Model() interface{} {
	return m.cfg
}

func (m *storageMock) MockLoad(err error) {
	m.mockedLoad = err
}

func (m *storageMock) MockSave(err error) {
	m.mockedSave = err
}
