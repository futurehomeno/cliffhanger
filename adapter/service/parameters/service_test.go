package parameters_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedparameters "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/parameters"
)

func TestService_SendParameterReport_Deduplicates(t *testing.T) {
	t.Parallel()

	controller := mockedparameters.NewController(t)
	controller.On("GetParameter", "brightness").Return(parameters.NewIntParameter("brightness", 10), nil).Twice()

	svc := newTestService(t, controller, 1)

	sent, err := svc.SendParameterReport("brightness", false)
	assert.NoError(t, err)
	assert.True(t, sent, "first report must be sent")

	sent, err = svc.SendParameterReport("brightness", false)
	assert.NoError(t, err)
	assert.False(t, sent, "unchanged parameter must not be reported again")
}

func TestService_SendParameterReport_ReportsChangedValue(t *testing.T) {
	t.Parallel()

	controller := mockedparameters.NewController(t)
	controller.On("GetParameter", "brightness").Return(parameters.NewIntParameter("brightness", 10), nil).Once()
	controller.On("GetParameter", "brightness").Return(parameters.NewIntParameter("brightness", 20), nil).Once()

	svc := newTestService(t, controller, 2)

	sent, err := svc.SendParameterReport("brightness", false)
	assert.NoError(t, err)
	assert.True(t, sent)

	sent, err = svc.SendParameterReport("brightness", false)
	assert.NoError(t, err)
	assert.True(t, sent, "changed parameter must be reported")
}

// TestService_SendParameterReport_IsKeyedPerParameter guards the sub key: caching every parameter
// under one key would let one parameter's value suppress or trigger another's report.
func TestService_SendParameterReport_IsKeyedPerParameter(t *testing.T) {
	t.Parallel()

	controller := mockedparameters.NewController(t)
	controller.On("GetParameter", "brightness").Return(parameters.NewIntParameter("brightness", 10), nil).Twice()
	controller.On("GetParameter", "contrast").Return(parameters.NewIntParameter("contrast", 99), nil).Once()

	svc := newTestService(t, controller, 2)

	sent, err := svc.SendParameterReport("brightness", false)
	assert.NoError(t, err)
	assert.True(t, sent)

	sent, err = svc.SendParameterReport("contrast", false)
	assert.NoError(t, err)
	assert.True(t, sent, "a different parameter must be reported on its own merit")

	sent, err = svc.SendParameterReport("brightness", false)
	assert.NoError(t, err)
	assert.False(t, sent, "reporting another parameter must not invalidate this one's cache entry")
}

// TestService_SendParameterReport_SurvivesControllerReuse covers a controller that hands back the
// same *Parameter on every call and mutates it in place. Caching that pointer would let the
// mutation rewrite the cached snapshot, so a genuine change would look unchanged and be suppressed.
func TestService_SendParameterReport_SurvivesControllerReuse(t *testing.T) {
	t.Parallel()

	reused := parameters.NewIntParameter("brightness", 10)

	controller := mockedparameters.NewController(t)
	controller.On("GetParameter", "brightness").Return(reused, nil).Twice()

	svc := newTestService(t, controller, 2)

	sent, err := svc.SendParameterReport("brightness", false)
	assert.NoError(t, err)
	assert.True(t, sent)

	mutated := parameters.NewIntParameter("brightness", 20)
	reused.Value = mutated.Value

	sent, err = svc.SendParameterReport("brightness", false)
	assert.NoError(t, err)
	assert.True(t, sent, "a change made in place by the controller must still be reported")
}

// TestService_SendSupportedParamsReport_SurvivesControllerReuse is the slice-shaped equivalent:
// the controller returns the same slice and mutates a specification within it.
func TestService_SendSupportedParamsReport_SurvivesControllerReuse(t *testing.T) {
	t.Parallel()

	reused := []*parameters.ParameterSpecification{
		{ID: "brightness", Name: "Brightness", ValueType: parameters.ValueTypeInt, WidgetType: parameters.WidgetTypeInput},
	}

	controller := mockedparameters.NewController(t)
	controller.On("GetParameterSpecifications").Return(reused, nil).Twice()

	svc := newTestService(t, controller, 2)

	sent, err := svc.SendSupportedParamsReport(false)
	assert.NoError(t, err)
	assert.True(t, sent)

	reused[0].Name = "Brightness level"

	sent, err = svc.SendSupportedParamsReport(false)
	assert.NoError(t, err)
	assert.True(t, sent, "a specification changed in place by the controller must still be reported")
}

func TestService_SendParameterReport_ForceBypassesCache(t *testing.T) {
	t.Parallel()

	controller := mockedparameters.NewController(t)
	controller.On("GetParameter", "brightness").Return(parameters.NewIntParameter("brightness", 10), nil).Twice()

	svc := newTestService(t, controller, 2)

	_, err := svc.SendParameterReport("brightness", true)
	assert.NoError(t, err)

	sent, err := svc.SendParameterReport("brightness", true)
	assert.NoError(t, err)
	assert.True(t, sent, "a forced report must bypass the cache")
}

func TestService_SendSupportedParamsReport_ForceBypassesCache(t *testing.T) {
	t.Parallel()

	specs := testSpecifications(t)

	controller := mockedparameters.NewController(t)
	controller.On("GetParameterSpecifications").Return(specs, nil).Twice()

	svc := newTestService(t, controller, 2)

	_, err := svc.SendSupportedParamsReport(true)
	assert.NoError(t, err)

	sent, err := svc.SendSupportedParamsReport(true)
	assert.NoError(t, err)
	assert.True(t, sent, "a forced report must bypass the cache")
}

func TestService_SendSupportedParamsReport_Deduplicates(t *testing.T) {
	t.Parallel()

	specs := testSpecifications(t)

	controller := mockedparameters.NewController(t)
	controller.On("GetParameterSpecifications").Return(specs, nil).Twice()

	svc := newTestService(t, controller, 1)

	sent, err := svc.SendSupportedParamsReport(false)
	assert.NoError(t, err)
	assert.True(t, sent, "first supported parameters report must be sent")

	sent, err = svc.SendSupportedParamsReport(false)
	assert.NoError(t, err)
	assert.False(t, sent, "unchanged supported parameters must not be reported again")
}

// newTestService builds a parameters service whose publisher expects exactly wantPublished
// messages, so a report that should have been suppressed fails the mock expectation too.
func newTestService(t *testing.T, controller parameters.Controller, wantPublished int) parameters.Service {
	t.Helper()

	publisher := mockedadapter.NewServicePublisher(t)
	publisher.On("PublishServiceMessage", mock.Anything, mock.Anything).Return(nil).Times(wantPublished)

	return parameters.NewService(publisher, &parameters.Config{
		Specification: parameters.Specification("test_adapter", "1", "2", nil),
		Controller:    controller,
	})
}
