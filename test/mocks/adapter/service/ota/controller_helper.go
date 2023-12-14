package mockedota

import (
	"github.com/futurehomeno/cliffhanger/adapter/service/ota"
)

func (_m *Controller) MockStartOTAUpdate(firmwarePath string, err error, once bool) *Controller {
	c := _m.On("StartOTAUpdate", firmwarePath).Return(err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockOTAStatusReport(report ota.StatusReport, err error, once bool) *Controller {
	c := _m.On("OTAStatusReport").Return(report, err)

	if once {
		c.Once()
	}

	return _m
}
