package han

import (
	"github.com/futurehomeno/cliffhanger/device"
)

// Meter is an interface representing a HAN Meter device.
type Meter interface {
	// GetReport returns simplified meter report based on input unit.
	GetReport(unit string) (float64, error)
	// GetExtendedReport returns extended meter report. Should return nil if extended report is not supported.
	GetExtendedReport() (map[string]float64, error)
	// GetSupportedUnits returns units that are supported by the simplified meter report.
	GetSupportedUnits() []string
	// GetSupportedExtendedValues returns extended values that are supported by the extended meter report.
	GetSupportedExtendedValues() []string
	// SupportsExtendedReport returns true if meter supports the extended report.
	SupportsExtendedReport() bool
}

// Provider is an interface representing a HAN meter provider to be used by an adapter implementation.
type Provider interface {
	// Register registers the device to be available under provided address.
	Register(address string, device Meter)
	// Unregister unregisters the device from the  provided address.
	Unregister(address string)
	// Get returns device registered at the provided address. If no device has been registered a nil value is returned.
	Get(address string) Meter
}

// NewProvider creates new instance of a HAN meter provider.
func NewProvider() Provider {
	return &provider{
		provider: device.NewProvider(),
	}
}

// provider is a private implementation of the HAN meter provider.
type provider struct {
	provider device.Provider
}

// Register registers the device to be available under provided address.
func (p *provider) Register(address string, device Meter) {
	p.provider.Register(address, device)
}

// Unregister unregisters the device from the  provided address.
func (p *provider) Unregister(address string) {
	p.provider.Unregister(address)
}

// Get returns device registered at the provided address. If no device has been registered a nil value is returned.
func (p *provider) Get(address string) Meter {
	d := p.provider.Get(address)
	if d == nil {
		return nil
	}

	meter, ok := d.(Meter)
	if !ok {
		return nil
	}

	return meter
}
