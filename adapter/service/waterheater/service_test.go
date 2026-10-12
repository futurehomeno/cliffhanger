package waterheater_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/futurehomeno/cliffhanger/adapter/service/waterheater"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedwaterheater "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/waterheater"
)

func TestService_SetpointReportCacheIsKeyedByNormalizedMode(t *testing.T) {
	t.Parallel()

	publisher := mockedadapter.NewServicePublisher(t)
	publisher.On("PublishServiceMessage", mock.Anything, mock.Anything).Return(nil).Once()

	svc := waterheater.NewService(publisher, &waterheater.Config{
		Specification: waterheater.Specification("test_adapter", "1", "2", nil, testModes, testSetpoints, testStates, testRange, nil, testStep),
		Controller: mockedwaterheater.NewController(t).
			MockWaterHeaterSetpointReport("heat", 65, "C", nil, false),
	})

	sent, err := svc.SendSetpointReport("HEAT", true)
	assert.NoError(t, err)
	assert.True(t, sent)

	sent, err = svc.SendSetpointReport("heat", false)
	assert.NoError(t, err)
	assert.False(t, sent, "an unchanged setpoint must not be re-reported under another case of its mode")
}

func TestService_SetpointReportIsRepublishedOnUnitChange(t *testing.T) {
	t.Parallel()

	publisher := mockedadapter.NewServicePublisher(t)
	publisher.On("PublishServiceMessage", mock.Anything, mock.Anything).Return(nil).Twice()

	svc := waterheater.NewService(publisher, &waterheater.Config{
		Specification: waterheater.Specification("test_adapter", "1", "2", nil, testModes, testSetpoints, testStates, testRange, nil, testStep),
		Controller: mockedwaterheater.NewController(t).
			MockWaterHeaterSetpointReport("heat", 65, "C", nil, true).
			MockWaterHeaterSetpointReport("heat", 65, "F", nil, true),
	})

	sent, err := svc.SendSetpointReport("heat", false)
	assert.NoError(t, err)
	assert.True(t, sent)

	sent, err = svc.SendSetpointReport("heat", false)
	assert.NoError(t, err)
	assert.True(t, sent, "a setpoint whose unit changed must be reported again")
}
