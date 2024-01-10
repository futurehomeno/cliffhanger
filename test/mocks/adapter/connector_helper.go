package mockedadapter

import (
	"testing"

	"github.com/futurehomeno/cliffhanger/adapter"
)

func NewDefaultConnector(t *testing.T) *Connector {
	return NewConnector(t).MockConnectivity(&adapter.ConnectivityDetails{
		ConnectionStatus:  adapter.ConnectionStatusUp,
		Operationability:  nil,
		ConnectionQuality: adapter.ConnectionQualityUndefined,
		ConnectionType:    adapter.ConnectionTypeIndirect,
	}, false)
}

func (_m *Connector) MockConnectivity(details *adapter.ConnectivityDetails, once bool) *Connector {
	c := _m.On("Connectivity").Return(details)

	if once {
		c.Once()
	} else {
		c.Maybe()
	}

	return _m
}
