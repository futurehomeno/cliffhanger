package scenectrl_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/adapter/service/scenectrl"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedscenectrl "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/scenectrl"
)

func TestService_SendSceneReport_SameSceneActivatedAgain(t *testing.T) {
	t.Parallel()

	first := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	controller := mockedscenectrl.NewController(t).
		MockSceneCtrlSceneReport(scenectrl.SceneReport{Scene: "movie", Timestamp: first}, nil, true).
		MockSceneCtrlSceneReport(scenectrl.SceneReport{Scene: "movie", Timestamp: first}, nil, true).
		MockSceneCtrlSceneReport(scenectrl.SceneReport{Scene: "movie", Timestamp: first.Add(time.Minute)}, nil, true)

	publisher := mockedadapter.NewServicePublisher(t)
	publisher.On("PublishServiceMessage", mock.Anything, mock.MatchedBy(func(m *fimpgo.FimpMessage) bool {
		return m.Interface == scenectrl.EvtSceneReport && m.ValueType == "string" && m.Value == "movie"
	})).Return(nil).Twice()

	svc := scenectrl.NewService(publisher, &scenectrl.Config{
		Specification: scenectrl.Specification("test_adapter", "1", "2", []string{"movie"}, nil),
		Controller:    controller,
	})

	for i, want := range []bool{true, false, true} {
		sent, err := svc.SendSceneReport(false)
		require.NoError(t, err)
		assert.Equal(t, want, sent, "call %d", i)
	}
}

func TestService_SetSceneRequiresExactSupportedScene(t *testing.T) {
	t.Parallel()

	svc := scenectrl.NewService(nil, &scenectrl.Config{
		Specification: scenectrl.Specification("test_adapter", "1", "2", []string{"movie"}, nil),
		Controller:    mockedscenectrl.NewController(t),
	})

	assert.ErrorContains(t, svc.SetScene("MOVIE"), "not supported")
}
