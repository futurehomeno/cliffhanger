package mockedadapter

import (
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/futurehomeno/cliffhanger/adapter"
)

func (_m *ControllableConnector) MockConnect() *ControllableConnector {
	_m.On("Connect", mock.Anything).Return()

	return _m
}

func (_m *ControllableConnector) MockDisconnect() *ControllableConnector {
	_m.On("Disconnect", mock.Anything).Return()

	return _m
}

func (_m *ControllableConnector) MockConnectivity(details *adapter.ConnectivityDetails, once bool) *ControllableConnector {
	c := _m.On("Connectivity").Return(details)

	if once {
		c.Once()
	} else {
		c.Maybe()
	}

	return _m
}

func NewDefaultControllableConnector(t *testing.T) *ControllableConnector {
	t.Helper()
	return NewControllableConnector(t).MockConnect().MockConnectivity(&adapter.ConnectivityDetails{
		ConnectionStatus:  adapter.ConnectionStatusUp,
		Operationability:  nil,
		ConnectionQuality: adapter.ConnectionQualityUndefined,
		ConnectionType:    adapter.ConnectionTypeIndirect,
	}, false)
}
