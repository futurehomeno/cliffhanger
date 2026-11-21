package chargepoint_test

import (
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/utils"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedchargepoint "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/chargepoint"
)

type cableLockAwareCharger struct {
	*mockedchargepoint.Controller
	*mockedchargepoint.CableLockAwareController
}

func TestService_CableLockAwareWithoutAdjustment(t *testing.T) {
	t.Parallel()

	aware := mockedchargepoint.NewCableLockAwareController(t)
	aware.On("ChargepointCableLockReport").Return(&chargepoint.CableReport{CableLock: true, CableCurrent: utils.Ptr(16)}, nil).Once()

	publisher := mockedadapter.NewServicePublisher(t)
	publisher.On("PublishServiceMessage", mock.Anything, mock.MatchedBy(func(m *fimpgo.FimpMessage) bool {
		return m.Interface == chargepoint.EvtCableLockReport && m.Properties[chargepoint.PropertyCableCurrent] == "16"
	})).Return(nil).Once()

	svc := chargepoint.NewService(publisher, &chargepoint.Config{
		Specification: chargepoint.Specification("test_adapter", "1", "2", nil, []chargepoint.State{chargepoint.StateReadyToCharge}),
		Controller:    &cableLockAwareCharger{Controller: mockedchargepoint.NewController(t), CableLockAwareController: aware},
	})

	assert.True(t, svc.IsCableLockAware())
	assert.False(t, svc.SupportsAdjustingCableLock())

	var msgTypes []string
	for _, i := range svc.Specification().Interfaces {
		msgTypes = append(msgTypes, i.MsgType)
	}

	assert.Contains(t, msgTypes, chargepoint.CmdCableLockGetReport)
	assert.Contains(t, msgTypes, chargepoint.EvtCableLockReport)
	assert.NotContains(t, msgTypes, chargepoint.CmdCableLockSet)

	assert.ErrorContains(t, svc.SetCableLock(true), "adjusting cable lock is not supported")

	sent, err := svc.SendCableLockReport(true)
	require.NoError(t, err)
	assert.True(t, sent)
}
