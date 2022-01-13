package device

import (
	"sync"
)

// Provider is an interface representing a generic device provider to be used by an adapter implementation.
type Provider interface {
	// Register registers the device to be available under provided address.
	Register(address string, device interface{})
	// Unregister unregisters the device from the  provided address.
	Unregister(address string)
	// Get returns device registered at the provided address. If no device has been registered a nil value is returned.
	Get(address string) interface{}
}

// NewProvider creates new instance of a generic device provider.
func NewProvider() Provider {
	return &provider{
		lock:    &sync.RWMutex{},
		devices: make(map[string]interface{}),
	}
}

// provider is a private implementation of the device provider.
type provider struct {
	lock    *sync.RWMutex
	devices map[string]interface{}
}

// Register registers the device to be available under provided address.
func (p *provider) Register(address string, device interface{}) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.devices[address] = device
}

// Unregister unregisters the device from the  provided address.
func (p *provider) Unregister(address string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	delete(p.devices, address)
}

// Get returns device registered at the provided address. If no device has been registered a nil value is returned.
func (p *provider) Get(address string) interface{} {
	p.lock.RLock()
	defer p.lock.RUnlock()

	device, ok := p.devices[address]
	if !ok {
		return nil
	}

	return device
}
