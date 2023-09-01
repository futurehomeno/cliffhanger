package mockedparameters

import (
	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
)

func (_m *Controller) MockGetParameter(paramID string, param parameters.Parameter, err error, once bool) *Controller {
	c := _m.On("GetParameter", paramID).Return(param, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockGetParameterSpecifications(specs []parameters.ParameterSpecification, err error, once bool) *Controller {
	c := _m.On("GetParameterSpecifications").Return(specs, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockSetParameter(param parameters.Parameter, err error, once bool) *Controller {
	c := _m.On("SetParameter", param).Return(err)

	if once {
		c.Once()
	}

	return _m
}
