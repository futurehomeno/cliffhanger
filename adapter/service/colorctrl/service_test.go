package colorctrl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter/service/colorctrl"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedcolorctrl "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/colorctrl"
)

func TestService_SupportedProperties(t *testing.T) {
	t.Parallel()

	cfg := &colorctrl.Config{
		Specification: colorctrl.Specification(
			"test_adapter",
			"1",
			"2",
			nil,
			[]string{"red", "green", "blue"},
			map[string]int{"default": 500},
		),
		Controller: mockedcolorctrl.NewController(t),
	}

	svc := colorctrl.NewService(mockedadapter.NewServicePublisher(t), cfg)

	assert.Equal(t, []string{"red", "green", "blue"}, svc.SupportedComponents())
	assert.Equal(t, map[string]int{"default": 500}, svc.Specification().Props[colorctrl.PropertySupportedDurations])
}
