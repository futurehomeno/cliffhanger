package outlvlswitch_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/utils"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedoutlvlswitch "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/outlvlswitch"
)

type transitionSwitch struct {
	*mockedoutlvlswitch.Controller
	*mockedoutlvlswitch.LevelTransitionController
}

func newSwitch(publisher adapter.ServicePublisher, controller outlvlswitch.Controller, options ...adapter.SpecificationOption) outlvlswitch.Service {
	return outlvlswitch.NewService(publisher, &outlvlswitch.Config{
		Specification: outlvlswitch.Specification("test_adapter", "1", "2", outlvlswitch.SwitchTypeOnAndOff, 100, 0, nil, options...),
		Controller:    controller,
	})
}

func TestService_SendLevelReport_PublishesLevelEvents(t *testing.T) {
	t.Parallel()

	controller := mockedoutlvlswitch.NewController(t).
		MockLevelSwitchLevelReport(40, nil, false)

	var events []*outlvlswitch.LevelEvent

	publisher := mockedadapter.NewServicePublisher(t)
	publisher.On("PublishServiceMessage", mock.Anything, mock.Anything).Return(errors.New("broker down")).Once()
	publisher.On("PublishServiceMessage", mock.Anything, mock.Anything).Return(nil).Once()
	publisher.On("PublishServiceEvent", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		events = append(events, args.Get(1).(*outlvlswitch.LevelEvent)) //nolint:forcetypeassert
	})

	svc := newSwitch(publisher, controller)

	_, err := svc.SendLevelReport(false)
	require.Error(t, err)
	assert.Empty(t, events, "a failed publication must not emit a level event")

	sent, err := svc.SendLevelReport(false)
	require.NoError(t, err)
	assert.True(t, sent)

	sent, err = svc.SendLevelReport(false)
	require.NoError(t, err)
	assert.False(t, sent)

	require.Len(t, events, 2)
	assert.True(t, events[0].HasChanged())
	assert.False(t, events[1].HasChanged())
	assert.Equal(t, 40, events[1].Level)
}

func TestService_SetLevelClampsIntoRange(t *testing.T) {
	t.Parallel()

	controller := mockedoutlvlswitch.NewController(t).
		MockSetLevelSwitchLevel(100, 0, nil, true).
		MockSetLevelSwitchLevel(0, 2*time.Second, nil, true)

	svc := newSwitch(nil, controller)

	require.NoError(t, svc.SetLevel(150, nil))
	require.NoError(t, svc.SetLevel(-5, utils.Ptr(2*time.Second)))
}

func TestService_StartLevelTransition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		options []adapter.SpecificationOption
		params  outlvlswitch.LevelTransitionParams
		want    outlvlswitch.LevelTransitionParams
	}{
		{
			name:   "unsupported parameters are dropped",
			params: outlvlswitch.LevelTransitionParams{StartLvl: utils.Ptr(10), Duration: utils.Ptr(5 * time.Second)},
			want:   outlvlswitch.LevelTransitionParams{},
		},
		{
			name:    "supported parameters are passed on",
			options: []adapter.SpecificationOption{outlvlswitch.WithSupportedDuration(), outlvlswitch.WithSupportedStartLevel()},
			params:  outlvlswitch.LevelTransitionParams{StartLvl: utils.Ptr(10), Duration: utils.Ptr(5 * time.Second)},
			want:    outlvlswitch.LevelTransitionParams{StartLvl: utils.Ptr(10), Duration: utils.Ptr(5 * time.Second)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			transition := mockedoutlvlswitch.NewLevelTransitionController(t)
			transition.On("StartLevelTransition", outlvlswitch.TransitionUp, tt.want).Return(nil).Once()

			svc := newSwitch(nil, &transitionSwitch{Controller: mockedoutlvlswitch.NewController(t), LevelTransitionController: transition}, tt.options...)

			require.NoError(t, svc.StartLevelTransition(outlvlswitch.TransitionUp, tt.params))
		})
	}
}

func TestService_StartLevelTransitionRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	transitionCtrl := &transitionSwitch{
		Controller:                mockedoutlvlswitch.NewController(t),
		LevelTransitionController: mockedoutlvlswitch.NewLevelTransitionController(t),
	}

	svc := newSwitch(nil, transitionCtrl, outlvlswitch.WithSupportedStartLevel())

	assert.ErrorContains(t, svc.StartLevelTransition("sideways", outlvlswitch.LevelTransitionParams{}), "incorrect value")
	assert.ErrorContains(t, svc.StartLevelTransition(outlvlswitch.TransitionDown, outlvlswitch.LevelTransitionParams{StartLvl: utils.Ptr(101)}), "out of the supported range")

	plain := newSwitch(nil, mockedoutlvlswitch.NewController(t))

	assert.ErrorContains(t, plain.StartLevelTransition(outlvlswitch.TransitionUp, outlvlswitch.LevelTransitionParams{}), "isn't supported")
	assert.ErrorContains(t, plain.StopLevelTransition(), "isn't supported")
}
