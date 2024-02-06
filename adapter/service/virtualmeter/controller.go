package virtualmeter

import (
	"fmt"

	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
)

type (
	// controller is a virtual meter elect controller that is injected into the numericmeter.Service.
	controller struct {
		vvm   *manager
		topic string
	}
)

var (
	_ numericmeter.Reporter           = (*controller)(nil)
	_ numericmeter.ResettableReporter = (*controller)(nil)
)

func newController(topic string, vvm *manager) numericmeter.Reporter {
	return &controller{
		vvm:   vvm,
		topic: topic,
	}
}

// MeterReport returns a report on the energy calculation done by the virtual meter.
func (c *controller) MeterReport(unit numericmeter.Unit) (float64, error) {
	if c.vvm == nil {
		return 0, fmt.Errorf("controller: virtual meter report failed, virtual meter manager isn't initialised")
	}

	value, err := c.vvm.report(c.topic, unit)
	if err != nil {
		return 0, fmt.Errorf("controller: failed to get virtual report by address - %s, for unit - %s. %w", c.topic, unit.String(), err)
	}

	return value, nil
}

func (c *controller) MeterReset() error {
	if c.vvm == nil {
		return fmt.Errorf("controller: virtual meter reset failed, virtual meter manager isn't initialised")
	}

	err := c.vvm.reset(c.topic)
	if err != nil {
		return fmt.Errorf("controller: failed to reset virtual meter by address - %s. %w", c.topic, err)
	}

	return nil
}
