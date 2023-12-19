package virtualmeter

import (
	"fmt"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
)

type (
	controller struct {
		vvm   Manager
		topic string
	}
)

var (
	_ numericmeter.Reporter = &controller{}
)

func newController(topic string, vvm Manager) numericmeter.Reporter {
	return &controller{
		vvm:   vvm,
		topic: topic,
	}
}

func (c *controller) MeterReport(unit numericmeter.Unit) (float64, error) {
	if c.vvm == nil {
		return 0, fmt.Errorf("virtual meter report failed, virtual meter manager isn't initialised.")
	}

	// TODO change back to unit
	value, err := c.vvm.Report(c.topic, unit.String())
	if err != nil {
		return 0, fmt.Errorf("failed to get virtual report by address: %s, for unit: %s. %w", c.topic, unit.String(), err)
	}

	return value, nil
}
