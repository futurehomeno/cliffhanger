package mockedadapter

import (
	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/mock"
)

func (t *Thing) WithSendInclusionReport(force, once, result bool, err error) *Thing {
	c := t.On("SendInclusionReport", force).Return(result, err)

	if once {
		c.Once()
	}

	return t
}

func (t *Thing) WithInclusionReported(report *fimptype.ThingInclusionReport, once bool) *Thing {
	c := t.On("InclusionReport").Return(report)

	if once {
		c.Once()
	}

	return t
}

func (t *Thing) WithAddress(addr string, once bool) *Thing {
	c := t.On("Address").Return(addr)

	if once {
		c.Once()
	}

	return t
}

func (t *Thing) WithServices(service fimptype.ServiceNameT, once bool, services []adapter.Service) *Thing {
	c := t.On("Services", service).Return(services)

	if once {
		c.Once()
	}

	return t
}

func (t *Thing) WithUpdate(once bool, err error) *Thing {
	c := t.On("Update", mock.Anything).Return(err)

	if once {
		c.Once()
	}

	return t
}
